// Package declarative implements the zero-code engine tier: engines configured
// entirely from YAML. The XPathEngine here is the Go port of
// searx/engines/xpath.py, which backs a large share of SearXNG's 200+ engines.
//
// Adding such an engine requires no Go and no Lua — just a YAML file declaring
// the search URL template and the XPath selectors.
package declarative

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/htmlx"
	"github.com/searxng/gosearx/internal/result"
)

// XPathConfig is the YAML schema for a declarative xpath engine. Field names and
// defaults mirror searx/engines/xpath.py.
type XPathConfig struct {
	Name   string `yaml:"name"`
	Engine string `yaml:"engine"` // must be "xpath"

	SearchURL string `yaml:"search_url"`
	Method    string `yaml:"method"`
	LangAll   string `yaml:"lang_all"`

	Paging                 bool  `yaml:"paging"`
	PageSize               int   `yaml:"page_size"`
	FirstPageNum           int   `yaml:"first_page_num"`
	SendPageNumOnFirstPage *bool `yaml:"send_page_num_on_first_page"`

	TimeRangeSupport bool           `yaml:"time_range_support"`
	TimeRangeURL     string         `yaml:"time_range_url"`
	TimeRangeMap     map[string]int `yaml:"time_range_map"`

	SafeSearch    bool           `yaml:"safesearch"`
	SafeSearchMap map[int]string `yaml:"safe_search_map"`

	ResultsXPath    string `yaml:"results_xpath"`
	URLXPath        string `yaml:"url_xpath"`
	TitleXPath      string `yaml:"title_xpath"`
	ContentXPath    string `yaml:"content_xpath"`
	ThumbnailXPath  string `yaml:"thumbnail_xpath"`
	SuggestionXPath string `yaml:"suggestion_xpath"`

	Categories []string `yaml:"categories"`
	Shortcut   string   `yaml:"shortcut"`
}

// XPathEngine is an engine.Engine driven by an XPathConfig.
type XPathEngine struct {
	cfg XPathConfig
}

// NewXPath validates and builds a declarative xpath engine.
func NewXPath(cfg XPathConfig) (*XPathEngine, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("xpath engine: name required")
	}
	if cfg.SearchURL == "" {
		return nil, fmt.Errorf("xpath engine %q: search_url required", cfg.Name)
	}
	// Defaults from xpath.py.
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
	if cfg.TimeRangeURL == "" {
		cfg.TimeRangeURL = "&hours={time_range_val}"
	}
	if cfg.TimeRangeMap == nil {
		cfg.TimeRangeMap = map[string]int{"day": 24, "week": 24 * 7, "month": 24 * 30, "year": 24 * 365}
	}
	if cfg.SafeSearchMap == nil {
		cfg.SafeSearchMap = map[int]string{0: "&filter=none", 1: "&filter=moderate", 2: "&filter=strict"}
	}
	return &XPathEngine{cfg: cfg}, nil
}

func (e *XPathEngine) Name() string { return e.cfg.Name }

// Request builds the search URL by substituting {query},{pageno},{lang},
// {time_range},{safe_search} into search_url (port of xpath.py request()).
func (e *XPathEngine) Request(_ context.Context, q engine.Query) (*engine.HTTPRequest, error) {
	c := e.cfg

	lang := c.LangAll
	if q.Locale != "all" && len(q.Locale) >= 2 {
		lang = q.Locale[:2]
	}

	timeRange := ""
	if c.TimeRangeSupport && q.TimeRange != "" {
		if val, ok := c.TimeRangeMap[q.TimeRange]; ok {
			timeRange = strings.ReplaceAll(c.TimeRangeURL, "{time_range_val}", strconv.Itoa(val))
		}
	}

	safeSearch := ""
	if c.SafeSearch {
		if v, ok := c.SafeSearchMap[q.SafeSearch]; ok {
			safeSearch = v
		}
	}

	pageno := ""
	if *c.SendPageNumOnFirstPage || q.PageNo != 1 {
		pageno = strconv.Itoa((q.PageNo-1)*c.PageSize + c.FirstPageNum)
	}

	repl := strings.NewReplacer(
		"{query}", url.QueryEscape(q.Query),
		"{pageno}", pageno,
		"{lang}", lang,
		"{time_range}", timeRange,
		"{safe_search}", safeSearch,
	)
	return &engine.HTTPRequest{
		Method:  c.Method,
		URL:     repl.Replace(c.SearchURL),
		Headers: map[string]string{},
		Cookies: map[string]string{},
	}, nil
}

// Response parses results via the configured XPath selectors (port of
// xpath.py response()).
func (e *XPathEngine) Response(_ context.Context, resp *engine.HTTPResponse) (result.EngineResults, error) {
	c := e.cfg
	out := result.EngineResults{}
	if len(resp.Body) == 0 {
		return out, nil
	}
	dom, err := htmlx.Parse(resp.Text())
	if err != nil {
		return nil, fmt.Errorf("xpath engine %q: parse: %w", c.Name, err)
	}

	if c.ResultsXPath != "" {
		nodes, err := htmlx.List(dom, c.ResultsXPath)
		if err != nil {
			return nil, fmt.Errorf("xpath engine %q results_xpath: %w", c.Name, err)
		}
		for _, n := range nodes {
			u, _ := htmlx.URL(n, c.URLXPath, c.SearchURL)
			title, _ := htmlx.Text(n, c.TitleXPath)
			content, _ := htmlx.Text(n, c.ContentXPath)
			if u == "" && title == "" {
				continue
			}
			mr := &result.MainResult{
				Type: result.TypeMain, EngineName: c.Name,
				URL: u, Title: title, Content: content,
			}
			if c.ThumbnailXPath != "" {
				if th, _ := htmlx.URL(n, c.ThumbnailXPath, c.SearchURL); th != "" {
					mr.Thumbnail = th
				}
			}
			out = append(out, mr)
		}
	} else {
		// Parallel-list form: zip url/title/content lists.
		urls, _ := htmlx.List(dom, c.URLXPath)
		titles, _ := htmlx.List(dom, c.TitleXPath)
		contents, _ := htmlx.List(dom, c.ContentXPath)
		for i := range urls {
			u := htmlx.ResolveURL(c.SearchURL, strings.TrimSpace(htmlx.NodeText(urls[i])))
			mr := &result.MainResult{Type: result.TypeMain, EngineName: c.Name, URL: u}
			if i < len(titles) {
				mr.Title = htmlx.NormalizeSpace(htmlx.NodeText(titles[i]))
			}
			if i < len(contents) {
				mr.Content = htmlx.NormalizeSpace(htmlx.NodeText(contents[i]))
			}
			out = append(out, mr)
		}
	}

	if c.SuggestionXPath != "" {
		sugg, _ := htmlx.List(dom, c.SuggestionXPath)
		for _, s := range sugg {
			out = append(out, &result.Suggestion{
				EngineName: c.Name,
				Value:      htmlx.NormalizeSpace(htmlx.NodeText(s)),
			})
		}
	}
	return out, nil
}
