// Package ai implements optional LLM answer synthesis: given a query and the
// top search results, it asks a pluggable LLM (Ollama by default, or any
// OpenAI-compatible endpoint) to write a concise, cited summary.
//
// Privacy: the default provider is a LOCAL Ollama, so no data leaves the host
// unless the operator explicitly configures a remote provider.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Source is one result fed to the model as citable context.
type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

// Service performs answer synthesis against a configured provider.
type Service struct {
	provider string
	baseURL  string
	model    string
	apiKey   string
	topN     int
	hc       *http.Client
}

// Config mirrors config.AIConfig (kept local to avoid an import cycle).
type Config struct {
	Provider string
	BaseURL  string
	Model    string
	APIKey   string
	TopN     int
	Timeout  time.Duration
}

// New builds a Service from config, applying defaults.
func New(c Config) *Service {
	if c.Provider == "" {
		c.Provider = "ollama"
	}
	if c.BaseURL == "" {
		if c.Provider == "ollama" {
			c.BaseURL = "http://localhost:11434"
		}
	}
	if c.Model == "" {
		c.Model = "llama3.2"
	}
	if c.TopN <= 0 {
		c.TopN = 5
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	return &Service{
		provider: c.Provider,
		baseURL:  strings.TrimRight(c.BaseURL, "/"),
		model:    c.Model,
		apiKey:   c.APIKey,
		topN:     c.TopN,
		hc:       &http.Client{Timeout: c.Timeout},
	}
}

// TopN is the configured number of sources to include.
func (s *Service) TopN() int { return s.topN }

// buildPrompt assembles the system+user messages with numbered sources so the
// model can cite them as [1], [2], ….
func (s *Service) buildPrompt(query string, sources []Source) (system, user string) {
	system = "You are a concise search assistant. Using ONLY the numbered sources " +
		"provided, write a short (2-4 sentence) direct answer to the user's query. " +
		"Cite sources inline using bracketed numbers like [1] or [2]. If the sources " +
		"do not contain the answer, say so briefly. Do not invent facts or URLs."

	var b strings.Builder
	fmt.Fprintf(&b, "Query: %s\n\nSources:\n", query)
	for i, src := range sources {
		content := src.Content
		if len(content) > 500 {
			content = content[:500]
		}
		fmt.Fprintf(&b, "[%d] %s\n%s\n%s\n\n", i+1, src.Title, src.URL, content)
	}
	b.WriteString("Answer:")
	return system, b.String()
}

// Synthesize streams the model's answer, invoking onDelta for each text chunk.
// Returns the full answer text. The sources slice is truncated to TopN.
func (s *Service) Synthesize(ctx context.Context, query string, sources []Source, onDelta func(string)) (string, error) {
	if len(sources) > s.topN {
		sources = sources[:s.topN]
	}
	if len(sources) == 0 {
		return "", fmt.Errorf("no sources to synthesize")
	}
	system, user := s.buildPrompt(query, sources)
	switch s.provider {
	case "openai":
		return s.streamOpenAI(ctx, system, user, onDelta)
	default:
		return s.streamOllama(ctx, system, user, onDelta)
	}
}

// ---- Ollama (native /api/chat streaming, newline-delimited JSON) ----

func (s *Service) streamOllama(ctx context.Context, system, user string, onDelta func(string)) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  s.model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: status %d", resp.StatusCode)
	}

	var full strings.Builder
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}
		if chunk.Message.Content != "" {
			full.WriteString(chunk.Message.Content)
			if onDelta != nil {
				onDelta(chunk.Message.Content)
			}
		}
		if chunk.Done {
			break
		}
	}
	return full.String(), sc.Err()
}

// ---- OpenAI-compatible (SSE streaming chat completions) ----

func (s *Service) streamOpenAI(ctx context.Context, system, user string, onDelta func(string)) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  s.model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, err := s.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai: status %d", resp.StatusCode)
	}

	var full strings.Builder
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			c := chunk.Choices[0].Delta.Content
			if c != "" {
				full.WriteString(c)
				if onDelta != nil {
					onDelta(c)
				}
			}
		}
	}
	return full.String(), sc.Err()
}
