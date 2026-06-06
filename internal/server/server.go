// Package server exposes the JSON HTTP API. Phase 1 implements:
//
//	GET /api/search?q=...&pageno=...&category=...&safesearch=...&time_range=...
//	GET /api/engines    -> list registered engines
//	GET /healthz
//
// The /api/search response is the typed contract the React frontend consumes.
package server

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/searxng/gosearx/internal/ai"
	"github.com/searxng/gosearx/internal/autocomplete"
	"github.com/searxng/gosearx/internal/bangs"
	"github.com/searxng/gosearx/internal/config"
	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/finance"
	"github.com/searxng/gosearx/internal/github"
	"github.com/searxng/gosearx/internal/metrics"
	"github.com/searxng/gosearx/internal/preferences"
	"github.com/searxng/gosearx/internal/proxy"
	"github.com/searxng/gosearx/internal/query"
	"github.com/searxng/gosearx/internal/result"
	"github.com/searxng/gosearx/internal/search"
)

// Server holds the dependencies for HTTP handlers.
type Server struct {
	reg      *engine.Registry
	searcher *search.Searcher
	// defaultCategory is used when the query selects none.
	defaultCategory string
	// assets is the embedded frontend filesystem (may be nil if not built).
	assets fs.FS
	// fin is the optional finance service (nil = finance disabled).
	fin *finance.Service
	// completer powers /api/autocomplete (nil = disabled).
	completer *autocomplete.Completer
	// imageProxy powers /image_proxy (nil = disabled).
	imageProxy *proxy.ImageProxy
	// favicon powers /favicon_proxy (nil = disabled).
	favicon *proxy.FaviconResolver
	// cfg is the instance configuration (for /api/config).
	cfg *config.Config
	// metrics powers /api/stats and /metrics (nil = disabled).
	metrics *metrics.Metrics
	// ai powers /api/answer LLM synthesis (nil = disabled).
	ai     *ai.Service
	aiAuto bool
}

// WithAI attaches the AI answer-synthesis service. auto runs it for every
// search; otherwise it runs only on explicit request.
func (s *Server) WithAI(svc *ai.Service, auto bool) *Server {
	s.ai = svc
	s.aiAuto = auto
	return s
}

// WithMetrics attaches the metrics tracker for stats endpoints.
func (s *Server) WithMetrics(m *metrics.Metrics) *Server {
	s.metrics = m
	return s
}

// WithAutocomplete attaches an autocomplete completer.
func (s *Server) WithAutocomplete(c *autocomplete.Completer) *Server {
	s.completer = c
	return s
}

// WithImageProxy attaches the image proxy.
func (s *Server) WithImageProxy(p *proxy.ImageProxy) *Server {
	s.imageProxy = p
	return s
}

// WithFavicon attaches the favicon resolver.
func (s *Server) WithFavicon(f *proxy.FaviconResolver) *Server {
	s.favicon = f
	return s
}

// WithFinance attaches a finance service; queries detected as ticker lookups
// get a chart/quote attached to the response.
func (s *Server) WithFinance(fin *finance.Service) *Server {
	s.fin = fin
	return s
}

// New builds a Server.
func New(reg *engine.Registry, searcher *search.Searcher) *Server {
	return &Server{reg: reg, searcher: searcher, defaultCategory: "general"}
}

// WithConfig attaches the instance config (used by /api/config and the UI).
func (s *Server) WithConfig(cfg *config.Config) *Server {
	s.cfg = cfg
	if cfg.Search.DefaultCategory != "" {
		s.defaultCategory = cfg.Search.DefaultCategory
	}
	return s
}

// WithAssets attaches an embedded frontend filesystem to be served at /.
func (s *Server) WithAssets(assets fs.FS) *Server {
	s.assets = assets
	return s
}

// Handler returns the configured http.Handler (routes).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/engines", s.handleEngines)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/search", s.handleSearch)
	mux.HandleFunc("POST /api/search", s.handleSearch) // privacy: POST queries
	mux.HandleFunc("GET /api/search/stream", s.handleSearchStream)
	mux.HandleFunc("GET /api/finance", s.handleFinance)
	mux.HandleFunc("GET /api/github/readme", s.handleGitHubReadme)
	mux.HandleFunc("GET /api/answer", s.handleAIAnswer)
	mux.HandleFunc("GET /api/preferences", s.handleGetPreferences)
	mux.HandleFunc("POST /api/preferences", s.handleSetPreferences)
	mux.HandleFunc("GET /clear_cookies", s.handleClearCookies)
	mux.HandleFunc("GET /opensearch.xml", s.handleOpenSearch)
	mux.HandleFunc("GET /api/about", s.handleAbout)
	if s.metrics != nil {
		mux.HandleFunc("GET /api/stats", s.handleStats)
		mux.HandleFunc("GET /metrics", s.handleOpenMetrics)
	}
	if s.completer != nil {
		mux.HandleFunc("GET /api/autocomplete", s.handleAutocomplete)
		mux.HandleFunc("GET /autocompleter", s.handleAutocomplete) // SearXNG-compatible path
	}
	if s.imageProxy != nil {
		mux.HandleFunc("GET /image_proxy", s.imageProxy.Serve)
	}
	if s.favicon != nil {
		mux.HandleFunc("GET /favicon_proxy", s.favicon.Serve)
	}
	if s.assets != nil {
		mux.Handle("GET /", s.spaHandler())
	}
	return mux
}

// spaHandler serves embedded static files, falling back to index.html for
// client-side routes (single-page app behavior).
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}
		// If the requested asset exists, serve it; else serve index.html.
		if _, err := fs.Stat(s.assets, path[1:]); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// SearchResponse is the JSON shape returned by /api/search.
type SearchResponse struct {
	Query         string                 `json:"query"`
	Pageno        int                    `json:"pageno"`
	Results       []*result.MainResult   `json:"results"`
	Answers       []*result.Answer       `json:"answers,omitempty"`
	Infoboxes     []*result.Infobox      `json:"infoboxes,omitempty"`
	Images        []*result.Image        `json:"images,omitempty"`
	Videos        []*result.Video        `json:"videos,omitempty"`
	Papers        []*result.Paper        `json:"papers,omitempty"`
	Torrents      []*result.Torrent      `json:"torrents,omitempty"`
	Maps          []*result.MapResult    `json:"maps,omitempty"`
	Codes         []*result.Code         `json:"codes,omitempty"`
	Files         []*result.File         `json:"files,omitempty"`
	KeyValues     []*result.KeyValue     `json:"keyvalues,omitempty"`
	Quotes        []*result.Quote        `json:"quotes,omitempty"`
	Charts        []*result.Chart        `json:"charts,omitempty"`
	GHRepos       []*result.GHRepo       `json:"ghRepos,omitempty"`
	GHCode        []*result.GHCode       `json:"ghCode,omitempty"`
	GHIssues      []*result.GHIssue      `json:"ghIssues,omitempty"`
	GHUsers       []*result.GHUser       `json:"ghUsers,omitempty"`
	GHTopics      []*result.GHTopic      `json:"ghTopics,omitempty"`
	GHCommits     []*result.GHCommit     `json:"ghCommits,omitempty"`
	GHDiscussions []*result.GHDiscussion `json:"ghDiscussions,omitempty"`
	// GHRanked is the unified, relevance-ranked GitHub result list (mixed types)
	// so the best match surfaces first regardless of type.
	GHRanked     []github.RankedItem   `json:"ghRanked,omitempty"`
	Suggestions  []string              `json:"suggestions,omitempty"`
	Corrections  []string              `json:"corrections,omitempty"`
	Timings      []search.Timing       `json:"timings,omitempty"`
	Unresponsive []search.Unresponsive `json:"unresponsive,omitempty"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	// Support both GET (?q=) and POST (form body) so the UI can use POST for
	// privacy (query not in the URL / server logs).
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
	}
	q := r.URL.Query()
	if r.Method == http.MethodPost {
		for k, v := range r.PostForm {
			if len(v) > 0 && q.Get(k) == "" {
				q.Set(k, v[0])
			}
		}
	}
	raw := q.Get("q")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing q parameter")
		return
	}
	prefs := preferences.FromRequest(r)

	// External bang (!!g foo) -> redirect straight to the external engine.
	// Resolved from the raw query before parsing strips bang tokens.
	if u := bangs.Resolve(raw); u != "" {
		http.Redirect(w, r, u, http.StatusFound)
		return
	}

	parsed := query.Parse(raw)

	// Categories from the ?categories= param (comma-separated) feed engine
	// selection alongside any !bang / :category selectors in the query text.
	for _, c := range splitCSV(q.Get("categories")) {
		parsed.Categories = append(parsed.Categories, c)
	}

	// First-class GitHub routing: when the github category is selected, detect
	// intent (repo / user / issue / pr / code / topic) and pick the right
	// GitHub engines + rewrite the query with qualifiers.
	searchText := parsed.Text
	if s.isGitHubSearch(parsed) {
		intent := github.Detect(parsed.Text)
		searchText = intent.Query
		var ghEngines []*engine.Registered
		for _, name := range intent.Engines {
			if reg := s.reg.ByName(name); reg != nil && !reg.Meta.Disabled {
				ghEngines = append(ghEngines, reg)
			}
		}
		if len(ghEngines) > 0 {
			s.runGitHubSearch(w, r, ghEngines, searchText, pagenoOf(q))
			return
		}
	}

	engines := s.resolveEngines(parsed)
	// Drop engines the user disabled in preferences.
	if len(prefs.DisabledEngines) > 0 {
		filtered := engines[:0]
		for _, e := range engines {
			if !prefs.IsEngineDisabled(e.Meta.Name) {
				filtered = append(filtered, e)
			}
		}
		engines = filtered
	}
	if len(engines) == 0 {
		writeError(w, http.StatusBadRequest, "no engines selected")
		return
	}

	pageno, _ := strconv.Atoi(q.Get("pageno"))
	if pageno < 1 {
		pageno = 1
	}
	// safesearch: query param wins, else preference, else config default.
	safe := s.cfgSafeSearch()
	if prefs.SafeSearch != nil {
		safe = *prefs.SafeSearch
	}
	if v := q.Get("safesearch"); v != "" {
		safe, _ = strconv.Atoi(v)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Kick off a finance lookup in parallel if the query looks like a ticker.
	// Default range is 1d (Google-style); the UI can refetch other ranges.
	chartCh := s.financeLookup(ctx, parsed.Text, finance.ParseRange(q.Get("range")))

	locale := orDefault(q.Get("locale"), prefs.Language)
	resp, err := s.searcher.Search(ctx, search.Query{
		Text:       parsed.Text,
		PageNo:     pageno,
		Locale:     orDefault(locale, "all"),
		SafeSearch: safe,
		TimeRange:  q.Get("time_range"),
		Engines:    engines,
		ClientIP:   clientIP(r),
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Merge any finance chart into the response (shown above web results).
	if chartCh != nil {
		if chart := <-chartCh; chart != nil {
			resp.Charts = append([]*result.Chart{chart}, resp.Charts...)
			if chart.Quote != nil {
				resp.Quotes = append([]*result.Quote{chart.Quote}, resp.Quotes...)
			}
		}
	}

	switch q.Get("format") {
	case "csv":
		writeCSV(w, resp.Results)
		return
	case "rss":
		writeRSS(w, parsed.Text, resp.Results)
		return
	}

	writeJSON(w, http.StatusOK, SearchResponse{
		Query:        parsed.Text,
		Pageno:       pageno,
		Results:      resp.Results,
		Answers:      resp.Answers,
		Infoboxes:    resp.Infoboxes,
		Images:       resp.Images,
		Videos:       resp.Videos,
		Papers:       resp.Papers,
		Torrents:     resp.Torrents,
		Maps:         resp.Maps,
		Codes:        resp.Codes,
		Files:        resp.Files,
		KeyValues:    resp.KeyValues,
		Quotes:       resp.Quotes,
		Charts:       resp.Charts,
		GHRepos:      resp.GHRepos,
		GHCode:       resp.GHCode,
		GHIssues:     resp.GHIssues,
		GHUsers:      resp.GHUsers,
		GHTopics:     resp.GHTopics,
		Suggestions:  resp.Suggestions,
		Corrections:  resp.Corrections,
		Timings:      resp.Timings,
		Unresponsive: resp.Unresponsive,
	})
}

// handleSearchStream runs a search and streams results to the client via
// Server-Sent Events, emitting each engine's results as it resolves so the page
// fills in progressively instead of waiting for the slowest engine.
func (s *Server) handleSearchStream(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	raw := q.Get("q")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing q parameter")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	prefs := preferences.FromRequest(r)
	parsed := query.Parse(raw)
	for _, c := range splitCSV(q.Get("categories")) {
		parsed.Categories = append(parsed.Categories, c)
	}
	engines := s.resolveEngines(parsed)
	if len(prefs.DisabledEngines) > 0 {
		filtered := engines[:0]
		for _, e := range engines {
			if !prefs.IsEngineDisabled(e.Meta.Name) {
				filtered = append(filtered, e)
			}
		}
		engines = filtered
	}
	if len(engines) == 0 {
		writeError(w, http.StatusBadRequest, "no engines selected")
		return
	}
	pageno := pagenoOf(q)
	safe := s.cfgSafeSearch()
	if prefs.SafeSearch != nil {
		safe = *prefs.SafeSearch
	}
	if v := q.Get("safesearch"); v != "" {
		safe, _ = strconv.Atoi(v)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	var mu sync.Mutex
	send := func(event string, payload any) {
		mu.Lock()
		defer mu.Unlock()
		b, _ := json.Marshal(payload)
		_, _ = w.Write([]byte("event: " + event + "\ndata: "))
		_, _ = w.Write(b)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}

	// Finance runs in parallel and streams its chart when ready.
	if chartCh := s.financeLookup(ctx, parsed.Text, finance.ParseRange(q.Get("range"))); chartCh != nil {
		go func() {
			if chart := <-chartCh; chart != nil {
				send("chart", chart)
			}
		}()
	}

	// Accumulate streamed chunks through the SAME container the non-streaming
	// path uses, so results are deduped across engines and scored by
	// engine-weight × positions. After each engine resolves we send a full
	// re-ranked SNAPSHOT (the client replaces its state), which keeps the
	// ordering correct while still filling in progressively.
	container := result.NewContainer()
	s.searcher.ConfigureContainer(container)
	for _, reg := range engines {
		container.SetEngineWeight(reg.Meta.Name, reg.Meta.Weight)
	}
	// Note: on_result plugin filtering is already applied per-chunk inside
	// SearchStream, so we don't double-filter here.

	s.searcher.SearchStream(ctx, search.Query{
		Text: parsed.Text, PageNo: pageno, Locale: orDefault(orDefault(q.Get("locale"), prefs.Language), "all"),
		SafeSearch: safe, TimeRange: q.Get("time_range"), Engines: engines,
		ClientIP: clientIP(r), UserAgent: r.UserAgent(),
	}, func(chunk search.EngineChunk) {
		if chunk.Err != nil {
			send("unresponsive", map[string]string{"engine": chunk.Engine, "reason": chunk.Err.Error()})
			return
		}
		container.AddFromEngine(chunk.Results)
		// Emit the current merged+ranked snapshot. The client replaces its
		// per-type arrays with this each time.
		snap := snapshotResponse(container)
		snap["engine"] = chunk.Engine
		snap["elapsed"] = chunk.Elapsed.Seconds()
		send("snapshot", snap)
	})

	send("done", map[string]bool{"done": true})
}

// snapshotResponse renders the container's current merged+ranked state into the
// per-type arrays the frontend expects.
//
// charts/quotes are intentionally omitted when empty: the finance chart arrives
// via a separate `chart` SSE event, and a snapshot carrying an empty "charts"
// array would clobber it on the client (replace semantics). Engine-produced
// charts/quotes (rare, from Lua engines) are still included when present.
func snapshotResponse(c *result.Container) map[string]any {
	m := map[string]any{
		"results":       c.Ordered(),
		"answers":       c.Answers(),
		"infoboxes":     c.Infoboxes(),
		"images":        c.Images(),
		"videos":        c.Videos(),
		"papers":        c.Papers(),
		"torrents":      c.Torrents(),
		"maps":          c.Maps(),
		"codes":         c.Codes(),
		"files":         c.Files(),
		"keyvalues":     c.KeyValues(),
		"suggestions":   c.Suggestions(),
		"corrections":   c.Corrections(),
		"ghRepos":       c.GHRepos(),
		"ghCode":        c.GHCode(),
		"ghIssues":      c.GHIssues(),
		"ghUsers":       c.GHUsers(),
		"ghTopics":      c.GHTopics(),
		"ghCommits":     c.GHCommits(),
		"ghDiscussions": c.GHDiscussions(),
	}
	if ch := c.Charts(); len(ch) > 0 {
		m["charts"] = ch
	}
	if qt := c.Quotes(); len(qt) > 0 {
		m["quotes"] = qt
	}
	return m
}

// resolveEngines turns parsed selectors into the engine set:
// explicit !shortcuts win; then :categories; else the default category.
func (s *Server) resolveEngines(p query.Parsed) []*engine.Registered {
	seen := map[string]bool{}
	var out []*engine.Registered
	add := func(reg *engine.Registered) {
		if reg != nil && !reg.Meta.Disabled && !seen[reg.Meta.Name] {
			seen[reg.Meta.Name] = true
			out = append(out, reg)
		}
	}

	for _, sc := range p.EngineShorts {
		add(s.reg.ByShortcut(sc))
	}
	for _, cat := range p.Categories {
		for _, reg := range s.reg.ByCategory(cat) {
			add(reg)
		}
	}
	// Only fall back to the default category when the user selected NOTHING
	// (no bang, no category). If they explicitly chose a category that has no
	// engines, respect that and return empty — never silently show general
	// results for an empty image/video/news tab.
	if len(out) == 0 && len(p.EngineShorts) == 0 && len(p.Categories) == 0 {
		for _, reg := range s.reg.ByCategory(s.defaultCategory) {
			add(reg)
		}
	}
	return out
}

// financeLookup returns a channel delivering a chart if the query is a ticker
// lookup, or nil if finance is disabled / no symbol detected.
func (s *Server) financeLookup(ctx context.Context, queryText string, rng finance.Range) <-chan *result.Chart {
	if s.fin == nil {
		return nil
	}
	symbol := finance.DetectSymbol(queryText)
	if symbol == "" {
		return nil
	}
	ch := make(chan *result.Chart, 1)
	go func() {
		chart, err := s.fin.ChartResult(ctx, symbol, rng)
		if err != nil || chart == nil || len(chart.Series) == 0 {
			ch <- nil
			return
		}
		ch <- chart
	}()
	return ch
}

// handleFinance serves a single chart for a symbol+range, used by the UI's range
// selector to refetch without re-running a full web search.
//
//	GET /api/finance?symbol=AAPL&range=1d
func (s *Server) handleFinance(w http.ResponseWriter, r *http.Request) {
	if s.fin == nil {
		writeError(w, http.StatusNotFound, "finance disabled")
		return
	}
	q := r.URL.Query()
	symbol := q.Get("symbol")
	if symbol == "" {
		// Allow a free-text query too (e.g. "bitcoin").
		symbol = finance.DetectSymbol(q.Get("q"))
	}
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "missing symbol")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	chart, err := s.fin.ChartResult(ctx, symbol, finance.ParseRange(q.Get("range")))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chart)
}

func (s *Server) handleEngines(w http.ResponseWriter, _ *http.Request) {
	type engineInfo struct {
		Name       string   `json:"name"`
		Shortcut   string   `json:"shortcut,omitempty"`
		Categories []string `json:"categories"`
		Disabled   bool     `json:"disabled"`
	}
	var infos []engineInfo
	for _, name := range s.reg.Names() {
		reg := s.reg.ByName(name)
		infos = append(infos, engineInfo{
			Name: reg.Meta.Name, Shortcut: reg.Meta.Shortcut,
			Categories: reg.Meta.Categories, Disabled: reg.Meta.Disabled,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"engines": infos})
}

// handleAutocomplete returns suggestions as ["query", ["s1","s2",...]] —
// the OpenSearch suggestions format SearXNG uses.
func (s *Server) handleAutocomplete(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusOK, []any{"", []string{}})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	sugg, err := s.completer.Complete(ctx, q)
	if err != nil {
		sugg = nil
	}
	if sugg == nil {
		sugg = []string{}
	}
	writeJSON(w, http.StatusOK, []any{q, sugg})
}

// handleConfig returns instance configuration the UI needs to render: the
// category tabs, search defaults, and feature toggles. Mirrors SearXNG's
// /config (subset).
func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	cats := []string{"general"}
	autocomplete, autocompleteMin := "", 4
	faviconOn := false
	safe := 0
	formats := []string{"json"}
	instanceName := "gosearx"
	financeOn := false
	if s.cfg != nil {
		if len(s.cfg.Search.CategoriesAsTabs) > 0 {
			cats = s.cfg.Search.CategoriesAsTabs
		}
		autocomplete = s.cfg.Search.Autocomplete
		if s.cfg.Search.AutocompleteMin > 0 {
			autocompleteMin = s.cfg.Search.AutocompleteMin
		}
		faviconOn = s.cfg.Search.FaviconResolver != ""
		safe = s.cfg.Search.SafeSearch
		if len(s.cfg.Search.Formats) > 0 {
			formats = s.cfg.Search.Formats
		}
		if s.cfg.General.InstanceName != "" {
			instanceName = s.cfg.General.InstanceName
		}
		financeOn = s.cfg.Finance.Enabled
	}

	// Engines available per category (so the UI can show what each tab queries).
	enginesByCat := map[string][]string{}
	for _, name := range s.reg.Names() {
		reg := s.reg.ByName(name)
		if reg.Meta.Disabled {
			continue
		}
		for _, c := range reg.Meta.Categories {
			enginesByCat[c] = append(enginesByCat[c], name)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"instance_name":       instanceName,
		"categories":          cats,
		"default_category":    s.defaultCategory,
		"autocomplete":        autocomplete,
		"autocomplete_min":    autocompleteMin,
		"favicon":             faviconOn,
		"safe_search":         safe,
		"formats":             formats,
		"finance":             financeOn,
		"ai":                  s.ai != nil,
		"ai_auto":             s.ai != nil && s.aiAuto,
		"engines_by_category": enginesByCat,
	})
}

// cfgSafeSearch returns the configured default safesearch level.
func (s *Server) cfgSafeSearch() int {
	if s.cfg != nil {
		return s.cfg.Search.SafeSearch
	}
	return 0
}

// handleGetPreferences returns the current preferences (from cookie).
func (s *Server) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, preferences.FromRequest(r))
}

// handleSetPreferences persists posted preferences to the cookie.
func (s *Server) handleSetPreferences(w http.ResponseWriter, r *http.Request) {
	var p preferences.Preferences
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid preferences")
		return
	}
	p.Write(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// handleClearCookies clears the preferences cookie.
func (s *Server) handleClearCookies(w http.ResponseWriter, _ *http.Request) {
	preferences.Clear(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// handleOpenSearch serves the OpenSearch descriptor so the instance can be added
// as a browser search engine.
func (s *Server) handleOpenSearch(w http.ResponseWriter, r *http.Request) {
	name := "gosearx"
	if s.cfg != nil && s.cfg.General.InstanceName != "" {
		name = s.cfg.General.InstanceName
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	base := scheme + "://" + r.Host
	w.Header().Set("Content-Type", "application/opensearchdescription+xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/" xmlns:moz="http://www.mozilla.org/2006/browser/search/">
  <ShortName>` + xmlEscape(name) + `</ShortName>
  <Description>` + xmlEscape(name) + ` metasearch</Description>
  <InputEncoding>UTF-8</InputEncoding>
  <Url type="text/html" method="get" template="` + base + `/?q={searchTerms}"/>
  <Url type="application/x-suggestions+json" template="` + base + `/api/autocomplete?q={searchTerms}"/>
  <Url type="application/rss+xml" method="get" template="` + base + `/api/search?q={searchTerms}&amp;format=rss"/>
  <moz:SearchForm>` + base + `/</moz:SearchForm>
</OpenSearchDescription>`))
}

// isGitHubSearch reports whether the query explicitly targets the github
// category (via :github or the ?categories=github param). Bare !gh bangs still
// go through the normal single-engine path.
func (s *Server) isGitHubSearch(p query.Parsed) bool {
	for _, c := range p.Categories {
		if c == "github" {
			return true
		}
	}
	return false
}

// runGitHubSearch executes the intent-selected GitHub engines and returns the
// relevance-ranked GitHub results (mixed types, best match first) as JSON.
func (s *Server) runGitHubSearch(w http.ResponseWriter, r *http.Request, engines []*engine.Registered, text string, pageno int) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	resp, err := s.searcher.Search(ctx, search.Query{
		Text: text, PageNo: pageno, Locale: "all", Engines: engines,
		ClientIP: clientIP(r), UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ranked := github.Rank(text, resp.GHRepos, resp.GHCode, resp.GHIssues,
		resp.GHUsers, resp.GHTopics, resp.GHCommits, resp.GHDiscussions)
	writeJSON(w, http.StatusOK, SearchResponse{
		Query: text, Pageno: pageno,
		GHRanked: ranked,
		GHRepos:  resp.GHRepos, GHCode: resp.GHCode, GHIssues: resp.GHIssues,
		GHUsers: resp.GHUsers, GHTopics: resp.GHTopics,
		GHCommits: resp.GHCommits, GHDiscussions: resp.GHDiscussions,
		Timings: resp.Timings, Unresponsive: resp.Unresponsive,
	})
}

// ghReadmeCache is a tiny in-process TTL cache for repo READMEs (small payloads,
// rarely changing) so repeated expand/collapse doesn't re-hit the GitHub API.
var (
	ghReadmeMu    sync.Mutex
	ghReadmeStore = map[string]ghReadmeEntry{}
)

type ghReadmeEntry struct {
	body string
	exp  time.Time
}

// githubToken returns the configured GitHub API token (from the github engine
// config), or "" if none is set.
func (s *Server) githubToken() string {
	if reg := s.reg.ByName("github"); reg != nil {
		return reg.Meta.Config["token"]
	}
	return ""
}

// handleGitHubReadme fetches a repo's README via the GitHub API and returns it
// as plain text for inline preview. Repo must be "owner/name".
func (s *Server) handleGitHubReadme(w http.ResponseWriter, r *http.Request) {
	repo := strings.TrimSpace(r.URL.Query().Get("repo"))
	// Validate owner/name to avoid SSRF / path injection into the API URL.
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(repo, " ?&#") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid repo"})
		return
	}

	ghReadmeMu.Lock()
	if e, ok := ghReadmeStore[repo]; ok && time.Now().Before(e.exp) {
		ghReadmeMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"readme": e.body})
		return
	}
	ghReadmeMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	url := "https://api.github.com/repos/" + parts[0] + "/" + parts[1] + "/readme"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	// Raw media type returns the decoded README body directly (no base64).
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if tok := s.githubToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"readme": ""})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusOK, map[string]string{"readme": ""})
		return
	}
	// Cap the body to keep previews lightweight (~256 KB).
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	md := string(body)

	ghReadmeMu.Lock()
	ghReadmeStore[repo] = ghReadmeEntry{body: md, exp: time.Now().Add(10 * time.Minute)}
	// Bound the map so it can't grow unbounded.
	if len(ghReadmeStore) > 256 {
		for k := range ghReadmeStore {
			delete(ghReadmeStore, k)
			if len(ghReadmeStore) <= 200 {
				break
			}
		}
	}
	ghReadmeMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"readme": md})
}

// handleAIAnswer runs a search, feeds the top results to the configured LLM, and
// streams a cited summary back via Server-Sent Events. Events:
//
//	event: sources -> [{title,url}, ...]   (the citations, sent first)
//	event: delta   -> {"text":"..."}        (incremental answer tokens)
//	event: done    -> {"answer":"..."}       (full text)
//	event: error   -> {"error":"..."}
func (s *Server) handleAIAnswer(w http.ResponseWriter, r *http.Request) {
	if s.ai == nil {
		writeError(w, http.StatusNotFound, "AI synthesis disabled")
		return
	}
	q := r.URL.Query()
	raw := q.Get("q")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing q parameter")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	parsed := query.Parse(raw)
	for _, c := range splitCSV(q.Get("categories")) {
		parsed.Categories = append(parsed.Categories, c)
	}
	engines := s.resolveEngines(parsed)
	if len(engines) == 0 {
		writeError(w, http.StatusBadRequest, "no engines selected")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()

	// Gather authoritative sources server-side (don't trust client-supplied text).
	resp, err := s.searcher.Search(ctx, search.Query{
		Text: parsed.Text, PageNo: 1, Locale: "all", Engines: engines,
		ClientIP: clientIP(r), UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	var mu sync.Mutex
	send := func(event string, payload any) {
		mu.Lock()
		defer mu.Unlock()
		b, _ := json.Marshal(payload)
		_, _ = w.Write([]byte("event: " + event + "\ndata: "))
		_, _ = w.Write(b)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}

	n := s.ai.TopN()
	sources := make([]ai.Source, 0, n)
	type cite struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	cites := make([]cite, 0, n)
	for _, mr := range resp.Results {
		if len(sources) >= n {
			break
		}
		sources = append(sources, ai.Source{Title: mr.Title, URL: mr.URL, Content: mr.Content})
		cites = append(cites, cite{Title: mr.Title, URL: mr.URL})
	}
	if len(sources) == 0 {
		send("error", map[string]string{"error": "no results to summarize"})
		return
	}
	send("sources", cites)

	answer, err := s.ai.Synthesize(ctx, parsed.Text, sources, func(delta string) {
		send("delta", map[string]string{"text": delta})
	})
	if err != nil {
		send("error", map[string]string{"error": err.Error()})
		return
	}
	send("done", map[string]string{"answer": answer})
}

func pagenoOf(q interface{ Get(string) string }) int {
	n, _ := strconv.Atoi(q.Get("pageno"))
	if n < 1 {
		return 1
	}
	return n
}

// handleStats returns per-engine timing/error/suspension stats as JSON.
func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"engines": s.metrics.Snapshot()})
}

// handleOpenMetrics exposes engine stats in Prometheus/OpenMetrics text format.
func (s *Server) handleOpenMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	var b strings.Builder
	b.WriteString("# HELP gosearx_engine_requests_total Total requests per engine\n")
	b.WriteString("# TYPE gosearx_engine_requests_total counter\n")
	for _, e := range s.metrics.Snapshot() {
		lbl := `{engine="` + e.Name + `"}`
		b.WriteString("gosearx_engine_requests_total" + lbl + " " + itoa64(e.Requests) + "\n")
		b.WriteString("gosearx_engine_errors_total" + lbl + " " + itoa64(e.Errors) + "\n")
		b.WriteString("gosearx_engine_avg_response_ms" + lbl + " " + ftoa(e.AvgTimeMS) + "\n")
	}
	_, _ = w.Write([]byte(b.String()))
}

// handleAbout returns instance metadata (name, engine count, version info).
func (s *Server) handleAbout(w http.ResponseWriter, _ *http.Request) {
	name := "gosearx"
	if s.cfg != nil && s.cfg.General.InstanceName != "" {
		name = s.cfg.General.InstanceName
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"instance_name": name,
		"engine_count":  len(s.reg.Names()),
		"engines":       s.reg.Names(),
		"about":         "gosearx — a Go metasearch engine with Lua/JS/exec engines & plugins, AI answer synthesis, and interactive finance charts.",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }
func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', 1, 64) }

// clientIP best-effort extracts the client IP (honoring X-Forwarded-For).
func clientIP(r *http.Request) string {
	// Proxy-set headers take precedence (Pangolin/Traefik forward the real IP).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// left-most entry is the original client
		first := strings.TrimSpace(strings.Split(xff, ",")[0])
		if first != "" {
			return first
		}
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		return xr
	}
	// Fall back to the socket peer; SplitHostPort handles IPv6 brackets.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// xmlEscape escapes a string for safe inclusion in XML text/attributes.
func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

// splitCSV splits a comma-separated list, trimming spaces and dropping empties.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
