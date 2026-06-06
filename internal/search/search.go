// Package search is the orchestrator: it runs the selected engines in parallel,
// each with its own timeout, collects results into a result.Container, and
// returns the merged/scored/ordered response. It is the Go port of
// searx/search/__init__.py's search_standard / search_multiple_requests, using
// goroutines + context instead of threads + thread-locals.
package search

import (
	"context"
	"strconv"
	"time"

	"github.com/searxng/gosearx/internal/cache"
	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/metrics"
	"github.com/searxng/gosearx/internal/network"
	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

// Query is the high-level search request handed to the orchestrator.
type Query struct {
	Text       string
	PageNo     int
	Locale     string
	SafeSearch int
	TimeRange  string
	// Engines is the resolved set of engines to query.
	Engines []*engine.Registered
	// TimeoutLimit optionally caps the overall search (0 = use engine timeouts).
	TimeoutLimit time.Duration
	// ClientIP and UserAgent describe the requester (passed to plugins).
	ClientIP  string
	UserAgent string
}

// Timing records how long an engine took.
type Timing struct {
	Engine string  `json:"engine"`
	Total  float64 `json:"total"` // seconds
}

// Unresponsive records an engine that errored or timed out.
type Unresponsive struct {
	Engine string `json:"engine"`
	Reason string `json:"reason"`
}

// Response is the orchestrator output.
type Response struct {
	Results       []*result.MainResult
	Answers       []*result.Answer
	Infoboxes     []*result.Infobox
	Images        []*result.Image
	Videos        []*result.Video
	Papers        []*result.Paper
	Torrents      []*result.Torrent
	Maps          []*result.MapResult
	Codes         []*result.Code
	Files         []*result.File
	KeyValues     []*result.KeyValue
	Quotes        []*result.Quote
	Charts        []*result.Chart
	GHRepos       []*result.GHRepo
	GHCode        []*result.GHCode
	GHIssues      []*result.GHIssue
	GHUsers       []*result.GHUser
	GHTopics      []*result.GHTopic
	GHCommits     []*result.GHCommit
	GHDiscussions []*result.GHDiscussion
	Suggestions   []string
	Corrections   []string
	Timings       []Timing
	Unresponsive  []Unresponsive
	// Aborted is true if a plugin's pre_search aborted the search.
	Aborted bool
}

// Searcher runs searches against a fetcher, optionally with plugins.
type Searcher struct {
	fetch    network.Fetcher
	plugins  *plugin.Storage
	metrics  *metrics.Metrics
	cache    cache.Cache
	cacheTTL time.Duration

	// collapseDups enables near-duplicate result folding.
	collapseDups bool
	// domainPenalties overrides the default SEO down-rank table (nil = default).
	domainPenalties map[string]float64
	// enrichSet records that enrichment was explicitly configured.
	enrichSet bool
}

// New returns a Searcher.
func New(f network.Fetcher) *Searcher { return &Searcher{fetch: f} }

// WithEnrichment configures result-quality passes: near-duplicate collapsing
// and SEO/content-farm domain penalties (nil penalties = built-in defaults).
func (s *Searcher) WithEnrichment(collapseDups bool, penalties map[string]float64) *Searcher {
	s.collapseDups = collapseDups
	s.domainPenalties = penalties
	s.enrichSet = true
	return s
}

// ConfigureContainer applies enrichment settings to a fresh container. Exposed
// so callers that build their own container (e.g. the SSE stream handler) get
// the same near-dup collapsing and SEO penalties.
func (s *Searcher) ConfigureContainer(c *result.Container) {
	if !s.enrichSet {
		return
	}
	c.SetCollapseDuplicates(s.collapseDups)
	if s.domainPenalties != nil {
		c.SetDomainPenalties(s.domainPenalties)
	}
}

// WithCache enables per-engine response caching (keyed by engine+query+params)
// with the given TTL, so repeat/pre-search queries skip the upstream fetch.
func (s *Searcher) WithCache(c cache.Cache, ttl time.Duration) *Searcher {
	s.cache = c
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	s.cacheTTL = ttl
	return s
}

// WithPlugins attaches a plugin storage whose hooks run around the search.
func (s *Searcher) WithPlugins(p *plugin.Storage) *Searcher {
	s.plugins = p
	return s
}

// WithMetrics attaches a metrics tracker (timings, errors, suspension).
func (s *Searcher) WithMetrics(m *metrics.Metrics) *Searcher {
	s.metrics = m
	return s
}

type engineOutput struct {
	name    string
	results result.EngineResults
	elapsed time.Duration
	err     error
}

// Search executes q against all its engines in parallel and merges results.
func (s *Searcher) Search(ctx context.Context, q Query) (*Response, error) {
	if q.PageNo == 0 {
		q.PageNo = 1
	}
	if q.Locale == "" {
		q.Locale = "all"
	}

	// Plugin pre_search: may abort the whole search.
	var sc *plugin.SearchContext
	if s.plugins != nil {
		sc = &plugin.SearchContext{Query: q.Text, ClientIP: q.ClientIP, UserAgent: q.UserAgent}
		if !s.plugins.PreSearch(sc) {
			return &Response{Aborted: true}, nil
		}
	}

	// Skip engines currently suspended due to repeated failures.
	active := q.Engines
	if s.metrics != nil {
		active = active[:0:0]
		for _, reg := range q.Engines {
			if !s.metrics.IsSuspended(reg.Meta.Name) {
				active = append(active, reg)
			}
		}
	}

	out := make(chan engineOutput, len(active))
	for _, reg := range active {
		go s.runEngine(ctx, reg, q, out)
	}

	container := result.NewContainer()
	s.ConfigureContainer(container)
	// Plugin on_result: filter/mutate each main result as it is ingested.
	if s.plugins != nil {
		container.SetOnResult(s.plugins.OnResult)
	}
	resp := &Response{}
	for range active {
		eo := <-out
		if eo.err != nil {
			resp.Unresponsive = append(resp.Unresponsive, Unresponsive{
				Engine: eo.name, Reason: eo.err.Error(),
			})
			if s.metrics != nil {
				s.metrics.RecordError(eo.name)
			}
			continue
		}
		container.AddFromEngine(eo.results)
		resp.Timings = append(resp.Timings, Timing{
			Engine: eo.name, Total: eo.elapsed.Seconds(),
		})
		if s.metrics != nil {
			s.metrics.RecordSuccess(eo.name, eo.elapsed)
		}
	}

	// Plugin post_search: append plugin-contributed results (answers, etc.).
	if s.plugins != nil && sc != nil {
		container.AddFromEngine(s.plugins.PostSearch(sc))
	}

	// Apply engine weights for scoring.
	for _, reg := range q.Engines {
		container.SetEngineWeight(reg.Meta.Name, reg.Meta.Weight)
	}

	resp.Results = container.Ordered()
	resp.Answers = container.Answers()
	resp.Infoboxes = container.Infoboxes()
	resp.Images = container.Images()
	resp.Videos = container.Videos()
	resp.Papers = container.Papers()
	resp.Torrents = container.Torrents()
	resp.Maps = container.Maps()
	resp.Codes = container.Codes()
	resp.Files = container.Files()
	resp.KeyValues = container.KeyValues()
	resp.Quotes = container.Quotes()
	resp.Charts = container.Charts()
	resp.GHRepos = container.GHRepos()
	resp.GHCode = container.GHCode()
	resp.GHIssues = container.GHIssues()
	resp.GHUsers = container.GHUsers()
	resp.GHTopics = container.GHTopics()
	resp.GHCommits = container.GHCommits()
	resp.GHDiscussions = container.GHDiscussions()
	resp.Suggestions = container.Suggestions()
	resp.Corrections = container.Corrections()
	return resp, nil
}

// EngineChunk is one engine's contribution, delivered as it resolves during a
// streaming search.
type EngineChunk struct {
	Engine  string
	Results result.EngineResults
	Elapsed time.Duration
	Err     error
}

// SearchStream runs the engines in parallel and invokes onChunk for each as it
// resolves (in completion order), enabling the server to stream results to the
// client instead of waiting for the slowest engine. It returns when all engines
// have reported or ctx is done. on_result plugin filtering is applied per chunk.
func (s *Searcher) SearchStream(ctx context.Context, q Query, onChunk func(EngineChunk)) {
	if q.PageNo == 0 {
		q.PageNo = 1
	}
	if q.Locale == "" {
		q.Locale = "all"
	}

	if s.plugins != nil {
		sc := &plugin.SearchContext{Query: q.Text, ClientIP: q.ClientIP, UserAgent: q.UserAgent}
		if !s.plugins.PreSearch(sc) {
			return
		}
		// Deliver plugin instant-answers first (cheap, no network).
		if added := s.plugins.PostSearch(sc); len(added) > 0 {
			onChunk(EngineChunk{Engine: "plugins", Results: added})
		}
	}

	active := q.Engines
	if s.metrics != nil {
		active = active[:0:0]
		for _, reg := range q.Engines {
			if !s.metrics.IsSuspended(reg.Meta.Name) {
				active = append(active, reg)
			}
		}
	}

	out := make(chan engineOutput, len(active))
	for _, reg := range active {
		go s.runEngine(ctx, reg, q, out)
	}

	for range active {
		select {
		case <-ctx.Done():
			return
		case eo := <-out:
			if eo.err != nil {
				if s.metrics != nil {
					s.metrics.RecordError(eo.name)
				}
				onChunk(EngineChunk{Engine: eo.name, Err: eo.err, Elapsed: eo.elapsed})
				continue
			}
			if s.metrics != nil {
				s.metrics.RecordSuccess(eo.name, eo.elapsed)
			}
			// Apply on_result plugin filtering to main results in the chunk.
			res := eo.results
			if s.plugins != nil {
				filtered := res[:0]
				for _, r := range res {
					if mr, ok := r.(*result.MainResult); ok {
						if !s.plugins.OnResult(mr) {
							continue
						}
					}
					filtered = append(filtered, r)
				}
				res = filtered
			}
			onChunk(EngineChunk{Engine: eo.name, Results: res, Elapsed: eo.elapsed})
		}
	}
}

// runEngine executes one engine with its own timeout-bounded context, then
// reports on the channel. Never panics out: a crashing engine is reported as an
// error, mirroring SearXNG isolating engine failures.
func (s *Searcher) runEngine(ctx context.Context, reg *engine.Registered, q Query, out chan<- engineOutput) {
	name := reg.Meta.Name
	defer func() {
		if r := recover(); r != nil {
			out <- engineOutput{name: name, err: panicErr(r)}
		}
	}()

	timeout := reg.Meta.Timeout
	if q.TimeoutLimit > 0 && q.TimeoutLimit < timeout {
		timeout = q.TimeoutLimit
	}
	ectx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	// Serve from the per-engine response cache if warm.
	cacheKey := engineCacheKey(name, q)
	if s.cache != nil {
		if blob, ok := s.cache.Get(ectx, cacheKey); ok {
			if res, err := result.Unmarshal([]byte(blob), name); err == nil {
				out <- engineOutput{name: name, results: res, elapsed: time.Since(start)}
				return
			}
		}
	}

	eq := engine.Query{
		Query: q.Text, PageNo: q.PageNo, Locale: q.Locale,
		SafeSearch: q.SafeSearch, TimeRange: q.TimeRange,
		Config: reg.Meta.Config,
	}

	req, err := reg.Engine.Request(ectx, eq)
	if err != nil {
		out <- engineOutput{name: name, err: err}
		return
	}
	if req == nil { // engine opted to skip this query
		out <- engineOutput{name: name, results: result.EngineResults{}, elapsed: time.Since(start)}
		return
	}

	httpResp, err := s.fetch.Fetch(ectx, req)
	if err != nil {
		out <- engineOutput{name: name, err: err}
		return
	}
	httpResp.Query = q.Text
	httpResp.Config = reg.Meta.Config

	results, err := reg.Engine.Response(ectx, httpResp)
	if err != nil {
		out <- engineOutput{name: name, err: err}
		return
	}
	// Cache successful, non-empty responses for reuse by repeat/pre-searches.
	if s.cache != nil && len(results) > 0 {
		if blob, err := result.Marshal(results); err == nil {
			s.cache.Set(ectx, cacheKey, string(blob), s.cacheTTL)
		}
	}
	out <- engineOutput{name: name, results: results, elapsed: time.Since(start)}
}

// engineCacheKey builds a stable cache key for an engine's response to a query.
func engineCacheKey(engineName string, q Query) string {
	return "eng:" + engineName + "|q:" + q.Text +
		"|p:" + strconv.Itoa(q.PageNo) +
		"|l:" + q.Locale +
		"|s:" + strconv.Itoa(q.SafeSearch) +
		"|t:" + q.TimeRange
}

type panicError struct{ v any }

func (e panicError) Error() string {
	if err, ok := e.v.(error); ok {
		return "panic: " + err.Error()
	}
	return "panic in engine"
}

func panicErr(v any) error { return panicError{v} }
