// Package engine defines the contract every search engine implements,
// regardless of tier (declarative YAML, Lua script, or native Go).
//
// This is the Go successor to SearXNG's "engine = Python module with
// request()/response() functions mutating a params dict". Here:
//   - request() -> Request(ctx, Query) (*HTTPRequest, error)  [returns, no mutation]
//   - response() -> Response(ctx, *HTTPResponse) (EngineResults, error)
package engine

import (
	"context"

	"github.com/searxng/gosearx/internal/result"
)

// Query is the per-search input handed to an engine's Request method.
// It is the read-only successor to the input half of SearXNG's params dict.
type Query struct {
	Query      string // user search terms
	PageNo     int    // 1-based page number
	Locale     string // SearXNG-style locale, e.g. "en-US" or "all"
	SafeSearch int    // 0=none, 1=moderate, 2=strict
	TimeRange  string // "", "day", "week", "month", "year"
	// Config carries per-engine settings (e.g. api_key, base_url) from
	// settings.yml, exposed to Lua engines as params.config.
	Config map[string]string
}

// HTTPRequest is what an engine's Request method produces. The engine never
// performs I/O itself (this keeps scripted engines pure and sandboxable); the
// host processor executes the request. Mirrors SearXNG's params["url"] etc.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Cookies map[string]string
	Body    []byte
}

// HTTPResponse is handed to an engine's Response method after the host fetches.
type HTTPResponse struct {
	StatusCode int
	URL        string
	Body       []byte
	// Query echoes the original search terms so engines that filter client-side
	// (e.g. icon sets) can access it in Response. Mirrors resp.search_params.
	Query string
	// Config echoes the per-engine settings so Response can branch on them
	// (e.g. ddg_category for the shared duckduckgo_extra engine).
	Config map[string]string
}

// Text returns the body as a string (convenience for HTML/JSON parsing).
func (r *HTTPResponse) Text() string { return string(r.Body) }

// Engine is the contract for an "online" engine (the common case). Offline
// engines (DB/command-backed) will get a separate interface in a later phase.
type Engine interface {
	// Name is the registered engine name (e.g. "google").
	Name() string
	// Request builds the HTTP request for the given query. Returning a nil
	// request with nil error means "skip this engine for this query"
	// (e.g. paging unsupported), mirroring SearXNG returning None.
	Request(ctx context.Context, q Query) (*HTTPRequest, error)
	// Response parses the fetched response into results.
	Response(ctx context.Context, resp *HTTPResponse) (result.EngineResults, error)
}
