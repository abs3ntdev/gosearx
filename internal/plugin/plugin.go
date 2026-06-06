// Package plugin implements the plugin tier: Lua scripts that hook into the
// search lifecycle. It is the Go/Lua successor to searx/plugins. Hooks mirror
// SearXNG:
//
//	pre_search(ctx)  -> bool   run before engines; false aborts the search
//	on_result(result) -> bool  per main result; false drops it, mutate in place
//	post_search(ctx) -> results  after engines; may append results
//
// Plugins may also declare keyword triggers; a keyword plugin only runs when the
// first query term matches one of its keywords.
package plugin

import "github.com/searxng/gosearx/internal/result"

// SearchContext is the read/writable context passed to pre/post hooks.
type SearchContext struct {
	Query string
	// ClientIP and UserAgent describe the requester (for self_info etc.).
	ClientIP  string
	UserAgent string
	// Added collects results contributed by post_search hooks.
	Added result.EngineResults
}

// Plugin is the interface implemented by every plugin (currently Lua-backed).
type Plugin interface {
	Name() string
	// Keywords, if non-empty, gate the plugin to queries whose first term
	// matches one of them.
	Keywords() []string
	PreSearch(sc *SearchContext) (allow bool, err error)
	OnResult(r *result.MainResult) (keep bool, err error)
	PostSearch(sc *SearchContext) (result.EngineResults, error)
}
