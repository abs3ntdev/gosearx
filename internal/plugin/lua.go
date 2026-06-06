package plugin

import (
	"fmt"
	"strings"

	golua "github.com/searxng/gosearx/internal/lua"
	"github.com/searxng/gosearx/internal/result"
	lua "github.com/yuin/gopher-lua"
)

// LuaPlugin is a Plugin backed by a Lua script. The script may define any of:
// pre_search(ctx), on_result(result), post_search(ctx), and an optional
// `keywords` global table.
type LuaPlugin struct {
	name     string
	pool     *golua.Pool
	keywords []string
}

// CompileLua parses a plugin script and reads its keyword triggers.
func CompileLua(name, src string) (*LuaPlugin, error) {
	pool, err := golua.Compile(name, src)
	if err != nil {
		return nil, err
	}
	p := &LuaPlugin{name: name, pool: pool}
	// Read keywords once from a throwaway state.
	L, err := pool.Acquire()
	if err != nil {
		return nil, err
	}
	if kw, ok := L.GetGlobal("keywords").(*lua.LTable); ok {
		kw.ForEach(func(_, v lua.LValue) {
			if s, ok := v.(lua.LString); ok {
				p.keywords = append(p.keywords, string(s))
			}
		})
	}
	pool.Release(L, true)
	return p, nil
}

func (p *LuaPlugin) Name() string       { return p.name }
func (p *LuaPlugin) Keywords() []string { return p.keywords }

// PreSearch calls pre_search(ctx) if defined. Missing hook = allow.
func (p *LuaPlugin) PreSearch(sc *SearchContext) (bool, error) {
	L, err := p.pool.Acquire()
	if err != nil {
		return true, err
	}
	healthy := true
	defer func() { p.pool.Release(L, healthy) }()

	fn := L.GetGlobal("pre_search")
	if fn == lua.LNil {
		return true, nil
	}
	ctx := L.NewTable()
	ctx.RawSetString("query", lua.LString(sc.Query))
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, ctx); err != nil {
		healthy = false
		return true, fmt.Errorf("plugin %s pre_search: %w", p.name, err)
	}
	ret := L.Get(-1)
	L.Pop(1)
	// nil/true => allow; explicit false => abort.
	return ret != lua.LFalse, nil
}

// OnResult calls on_result(result); mutations on the table propagate back.
func (p *LuaPlugin) OnResult(r *result.MainResult) (bool, error) {
	L, err := p.pool.Acquire()
	if err != nil {
		return true, err
	}
	healthy := true
	defer func() { p.pool.Release(L, healthy) }()

	fn := L.GetGlobal("on_result")
	if fn == lua.LNil {
		return true, nil
	}
	tbl := L.NewTable()
	tbl.RawSetString("title", lua.LString(r.Title))
	tbl.RawSetString("url", lua.LString(r.URL))
	tbl.RawSetString("content", lua.LString(r.Content))
	tbl.RawSetString("engine", lua.LString(r.EngineName))

	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, tbl); err != nil {
		healthy = false
		return true, fmt.Errorf("plugin %s on_result: %w", p.name, err)
	}
	ret := L.Get(-1)
	L.Pop(1)
	if ret == lua.LFalse {
		return false, nil // drop result
	}
	// Apply any mutations the plugin made to the table back onto the result.
	r.Title = optStr(tbl, "title", r.Title)
	r.URL = optStr(tbl, "url", r.URL)
	r.Content = optStr(tbl, "content", r.Content)
	return true, nil
}

// PostSearch calls post_search(ctx) and converts any returned result tables.
func (p *LuaPlugin) PostSearch(sc *SearchContext) (result.EngineResults, error) {
	L, err := p.pool.Acquire()
	if err != nil {
		return nil, err
	}
	healthy := true
	defer func() { p.pool.Release(L, healthy) }()

	fn := L.GetGlobal("post_search")
	if fn == lua.LNil {
		return nil, nil
	}
	ctx := L.NewTable()
	ctx.RawSetString("query", lua.LString(sc.Query))
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, ctx); err != nil {
		healthy = false
		return nil, fmt.Errorf("plugin %s post_search: %w", p.name, err)
	}
	ret := L.Get(-1)
	L.Pop(1)
	tbl, ok := ret.(*lua.LTable)
	if !ok {
		return nil, nil
	}
	var out result.EngineResults
	tbl.ForEach(func(_, v lua.LValue) {
		if row, ok := v.(*lua.LTable); ok {
			m, _ := golua.LuaToGo(row).(map[string]any)
			if m == nil {
				return
			}
			if r := result.FromMap(p.name, m); r != nil {
				out = append(out, r)
			}
		}
	})
	return out, nil
}

// Close releases pooled states.
func (p *LuaPlugin) Close() { p.pool.Close() }

func optStr(t *lua.LTable, key, def string) string {
	if v, ok := t.RawGetString(key).(lua.LString); ok {
		return string(v)
	}
	return def
}

// MatchKeyword reports whether the query's first term matches a plugin keyword.
func MatchKeyword(keywords []string, query string) bool {
	if len(keywords) == 0 {
		return true // not a keyword-gated plugin
	}
	first := query
	if i := strings.IndexByte(query, ' '); i >= 0 {
		first = query[:i]
	}
	for _, kw := range keywords {
		if strings.EqualFold(kw, first) {
			return true
		}
	}
	return false
}
