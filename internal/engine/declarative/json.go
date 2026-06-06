// json.go implements the declarative JSON engine: the zero-code tier for
// API-based engines that return JSON. Go port of searx/engines/json_engine.py.
package declarative

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/htmlx"
	"github.com/searxng/gosearx/internal/result"
)

// JSONConfig is the YAML schema for a declarative json engine. Mirrors
// json_engine.py configuration.
type JSONConfig struct {
	Name   string `yaml:"name"`
	Engine string `yaml:"engine"` // "json" / "json_engine"

	SearchURL string `yaml:"search_url"`
	Method    string `yaml:"method"`
	LangAll   string `yaml:"lang_all"`

	Paging                 bool  `yaml:"paging"`
	PageSize               int   `yaml:"page_size"`
	FirstPageNum           int   `yaml:"first_page_num"`
	SendPageNumOnFirstPage *bool `yaml:"send_page_num_on_first_page"`

	ResultsQuery    string `yaml:"results_query"`
	URLQuery        string `yaml:"url_query"`
	URLPrefix       string `yaml:"url_prefix"`
	TitleQuery      string `yaml:"title_query"`
	ContentQuery    string `yaml:"content_query"`
	ThumbnailQuery  string `yaml:"thumbnail_query"`
	ThumbnailPrefix string `yaml:"thumbnail_prefix"`
	SuggestionQuery string `yaml:"suggestion_query"`

	Categories []string `yaml:"categories"`
	Shortcut   string   `yaml:"shortcut"`
}

// JSONEngine is an engine.Engine driven by a JSONConfig.
type JSONEngine struct {
	cfg JSONConfig
}

// NewJSON validates and builds a declarative json engine.
func NewJSON(cfg JSONConfig) (*JSONEngine, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("json engine: name required")
	}
	if cfg.SearchURL == "" {
		return nil, fmt.Errorf("json engine %q: search_url required", cfg.Name)
	}
	if cfg.URLQuery == "" || cfg.TitleQuery == "" {
		return nil, fmt.Errorf("json engine %q: url_query and title_query required", cfg.Name)
	}
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.LangAll == "" {
		cfg.LangAll = "en"
	}
	if cfg.PageSize == 0 {
		cfg.PageSize = 1
	}
	if cfg.FirstPageNum == 0 {
		cfg.FirstPageNum = 1
	}
	if cfg.SendPageNumOnFirstPage == nil {
		b := true
		cfg.SendPageNumOnFirstPage = &b
	}
	return &JSONEngine{cfg: cfg}, nil
}

func (e *JSONEngine) Name() string { return e.cfg.Name }

// Request builds the search URL (port of json_engine.request()).
func (e *JSONEngine) Request(_ context.Context, q engine.Query) (*engine.HTTPRequest, error) {
	c := e.cfg
	lang := c.LangAll
	if q.Locale != "all" && len(q.Locale) >= 2 {
		lang = q.Locale[:2]
	}
	pageno := ""
	if *c.SendPageNumOnFirstPage || q.PageNo != 1 {
		pageno = strconv.Itoa((q.PageNo-1)*c.PageSize + c.FirstPageNum)
	}
	repl := strings.NewReplacer(
		"{query}", url.QueryEscape(q.Query),
		"{pageno}", pageno,
		"{lang}", lang,
		"{time_range}", "",
		"{safe_search}", "",
	)
	return &engine.HTTPRequest{
		Method:  c.Method,
		URL:     repl.Replace(c.SearchURL),
		Headers: map[string]string{"Accept": "application/json"},
		Cookies: map[string]string{},
	}, nil
}

// Response decodes JSON and extracts results via the slash-path queries
// (port of json_engine.response()).
func (e *JSONEngine) Response(_ context.Context, resp *engine.HTTPResponse) (result.EngineResults, error) {
	c := e.cfg
	out := result.EngineResults{}
	if len(resp.Body) == 0 {
		return out, nil
	}
	var data any
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return nil, fmt.Errorf("json engine %q: decode: %w", c.Name, err)
	}

	var rows []any
	if c.ResultsQuery != "" {
		// results_query points at the array(s) of result objects.
		for _, m := range jsonQuery(data, c.ResultsQuery) {
			if arr, ok := m.([]any); ok {
				rows = append(rows, arr...)
			} else {
				rows = append(rows, m)
			}
		}
	} else if arr, ok := data.([]any); ok {
		rows = arr
	}

	for _, row := range rows {
		u := toString(jsonQueryOne(row, c.URLQuery))
		if c.URLPrefix != "" && u != "" {
			u = c.URLPrefix + u
		}
		title := toString(jsonQueryOne(row, c.TitleQuery))
		if u == "" && title == "" {
			continue
		}
		mr := &result.MainResult{
			Type: result.TypeMain, EngineName: c.Name, URL: u,
			Title: htmlx.StripHTML(title),
		}
		if c.ContentQuery != "" {
			mr.Content = htmlx.SanitizeHTML(toString(jsonQueryOne(row, c.ContentQuery)))
		}
		if c.ThumbnailQuery != "" {
			th := toString(jsonQueryOne(row, c.ThumbnailQuery))
			if th != "" {
				mr.Thumbnail = c.ThumbnailPrefix + th
			}
		}
		out = append(out, mr)
	}

	if c.SuggestionQuery != "" {
		for _, s := range jsonQuery(data, c.SuggestionQuery) {
			// A match may be a scalar suggestion or an array of them.
			if arr, ok := s.([]any); ok {
				for _, item := range arr {
					if str := toString(item); str != "" {
						out = append(out, &result.Suggestion{EngineName: c.Name, Value: str})
					}
				}
				continue
			}
			if str := toString(s); str != "" {
				out = append(out, &result.Suggestion{EngineName: c.Name, Value: str})
			}
		}
	}
	return out, nil
}
