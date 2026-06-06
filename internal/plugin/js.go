package plugin

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/searxng/gosearx/internal/jsruntime"
	"github.com/searxng/gosearx/internal/result"
)

// JSPlugin is a Plugin backed by a JavaScript program (goja, pure-Go). The
// program may define any of: preSearch(ctx), onResult(result),
// postSearch(ctx), and an optional `keywords` global array.
//
// Hook names use camelCase (JS convention); snake_case aliases are also
// accepted so authors can copy the Lua style if they prefer.
type JSPlugin struct {
	name     string
	pool     *jsruntime.Pool
	keywords []string
}

// CompileJS parses a JS plugin program and reads its keyword triggers.
func CompileJS(name, src string) (*JSPlugin, error) {
	pool, err := jsruntime.Compile(name, src)
	if err != nil {
		return nil, err
	}
	p := &JSPlugin{name: name, pool: pool}
	rt, err := pool.Acquire()
	if err != nil {
		return nil, err
	}
	if v := rt.GlobalObject().Get("keywords"); v != nil && !goja.IsUndefined(v) {
		if arr, ok := v.Export().([]any); ok {
			for _, e := range arr {
				if s, ok := e.(string); ok {
					p.keywords = append(p.keywords, s)
				}
			}
		}
	}
	pool.Release(rt, true)
	return p, nil
}

func (p *JSPlugin) Name() string       { return p.name }
func (p *JSPlugin) Keywords() []string { return p.keywords }

// fn looks up a global function by any of the given names.
func fn(rt *goja.Runtime, names ...string) (goja.Callable, bool) {
	for _, n := range names {
		if v := rt.GlobalObject().Get(n); v != nil && !goja.IsUndefined(v) {
			if c, ok := goja.AssertFunction(v); ok {
				return c, true
			}
		}
	}
	return nil, false
}

// PreSearch calls preSearch(ctx); missing hook = allow.
func (p *JSPlugin) PreSearch(sc *SearchContext) (bool, error) {
	rt, err := p.pool.Acquire()
	if err != nil {
		return true, err
	}
	healthy := true
	defer func() { p.pool.Release(rt, healthy) }()

	call, ok := fn(rt, "preSearch", "pre_search")
	if !ok {
		return true, nil
	}
	ctx := rt.ToValue(map[string]any{
		"query": sc.Query, "client_ip": sc.ClientIP, "user_agent": sc.UserAgent,
	})
	out, err := call(goja.Undefined(), ctx)
	if err != nil {
		healthy = false
		return true, fmt.Errorf("plugin %s preSearch: %w", p.name, err)
	}
	// undefined/true => allow; explicit false => abort.
	if out != nil && !goja.IsUndefined(out) && !out.ToBoolean() {
		return false, nil
	}
	return true, nil
}

// OnResult calls onResult(result); a returned object mutates title/url/content.
func (p *JSPlugin) OnResult(r *result.MainResult) (bool, error) {
	rt, err := p.pool.Acquire()
	if err != nil {
		return true, err
	}
	healthy := true
	defer func() { p.pool.Release(rt, healthy) }()

	call, ok := fn(rt, "onResult", "on_result")
	if !ok {
		return true, nil
	}
	arg := rt.ToValue(map[string]any{
		"title": r.Title, "url": r.URL, "content": r.Content, "engine": r.EngineName,
	})
	out, err := call(goja.Undefined(), arg)
	if err != nil {
		healthy = false
		return true, fmt.Errorf("plugin %s onResult: %w", p.name, err)
	}
	if out == nil || goja.IsUndefined(out) {
		return true, nil
	}
	// false => drop; object => apply mutations; true => keep unchanged.
	if b, ok := out.Export().(bool); ok {
		return b, nil
	}
	if m, ok := out.Export().(map[string]any); ok {
		if v, ok := m["title"].(string); ok {
			r.Title = v
		}
		if v, ok := m["url"].(string); ok {
			r.URL = v
		}
		if v, ok := m["content"].(string); ok {
			r.Content = v
		}
	}
	return true, nil
}

// PostSearch calls postSearch(ctx) and converts the returned array of results.
func (p *JSPlugin) PostSearch(sc *SearchContext) (result.EngineResults, error) {
	rt, err := p.pool.Acquire()
	if err != nil {
		return nil, err
	}
	healthy := true
	defer func() { p.pool.Release(rt, healthy) }()

	call, ok := fn(rt, "postSearch", "post_search")
	if !ok {
		return nil, nil
	}
	ctx := rt.ToValue(map[string]any{
		"query": sc.Query, "client_ip": sc.ClientIP, "user_agent": sc.UserAgent,
	})
	out, err := call(goja.Undefined(), ctx)
	if err != nil {
		healthy = false
		return nil, fmt.Errorf("plugin %s postSearch: %w", p.name, err)
	}
	val := jsruntime.MarshalReply(rt, out)
	rows, ok := val.([]any)
	if !ok {
		return nil, nil
	}
	var res result.EngineResults
	for _, row := range rows {
		if m, ok := row.(map[string]any); ok {
			if r := result.FromMap(p.name, m); r != nil {
				res = append(res, r)
			}
		}
	}
	return res, nil
}

// Close drops pooled runtimes.
func (p *JSPlugin) Close() { p.pool.Close() }
