// Package autocomplete provides search-as-you-type suggestions by proxying to a
// configured backend (google, duckduckgo). Port of searx/autocomplete.py
// (subset). The frontend calls /api/autocomplete?q=...
package autocomplete

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Backend fetches completions for a partial query.
type Backend func(ctx context.Context, hc *http.Client, query string) ([]string, error)

var backends = map[string]Backend{
	"google":     googleBackend,
	"duckduckgo": duckduckgoBackend,
	"brave":      braveBackend,
}

// Completer wraps a backend + HTTP client.
type Completer struct {
	name    string
	backend Backend
	hc      *http.Client
}

// New returns a Completer for the named backend, or nil if name is empty/unknown.
func New(name string) *Completer {
	b, ok := backends[name]
	if !ok {
		return nil
	}
	return &Completer{name: name, backend: b, hc: &http.Client{Timeout: 3 * time.Second}}
}

// Complete returns suggestions for query.
func (c *Completer) Complete(ctx context.Context, query string) ([]string, error) {
	if query == "" {
		return nil, nil
	}
	return c.backend(ctx, c.hc, query)
}

// google returns suggestions via the suggestqueries endpoint, which responds
// with ["query", ["s1","s2",...]].
func googleBackend(ctx context.Context, hc *http.Client, query string) ([]string, error) {
	u := "https://suggestqueries.google.com/complete/search?client=firefox&q=" + url.QueryEscape(query)
	body, err := fetch(ctx, hc, u)
	if err != nil {
		return nil, err
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) < 2 {
		return nil, err
	}
	var sugg []string
	_ = json.Unmarshal(raw[1], &sugg)
	return sugg, nil
}

// duckduckgo returns suggestions via ac/ which responds with [{"phrase":...}].
func duckduckgoBackend(ctx context.Context, hc *http.Client, query string) ([]string, error) {
	u := "https://duckduckgo.com/ac/?type=list&q=" + url.QueryEscape(query)
	body, err := fetch(ctx, hc, u)
	if err != nil {
		return nil, err
	}
	// type=list returns ["query", ["s1","s2"]]
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) < 2 {
		return nil, err
	}
	var sugg []string
	_ = json.Unmarshal(raw[1], &sugg)
	return sugg, nil
}

// brave returns suggestions via its suggest endpoint: ["query",["s1",...]].
func braveBackend(ctx context.Context, hc *http.Client, query string) ([]string, error) {
	u := "https://search.brave.com/api/suggest?q=" + url.QueryEscape(query)
	body, err := fetch(ctx, hc, u)
	if err != nil {
		return nil, err
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) < 2 {
		return nil, err
	}
	var sugg []string
	_ = json.Unmarshal(raw[1], &sugg)
	return sugg, nil
}

func fetch(ctx context.Context, hc *http.Client, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (gosearx)")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("autocomplete: status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}
