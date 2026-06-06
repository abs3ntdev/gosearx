// Package jsscript implements the JavaScript engine tier using goja (pure Go,
// no CGO). It mirrors the Lua tier (internal/engine/script): an engine program
// defines request(query, params) and response(resp) functions, sharing the same
// host contract (engine.Engine) so the orchestrator is tier-blind.
package jsscript

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/jsruntime"
	"github.com/searxng/gosearx/internal/result"
)

// JSEngine is an engine.Engine backed by a JavaScript program.
type JSEngine struct {
	name string
	pool *jsruntime.Pool
}

// Compile parses a JS engine program once and returns a ready JSEngine.
func Compile(name, src string) (*JSEngine, error) {
	pool, err := jsruntime.Compile(name, src)
	if err != nil {
		return nil, err
	}
	return &JSEngine{name: name, pool: pool}, nil
}

func (e *JSEngine) Name() string { return e.name }

func callable(rt *goja.Runtime, name string) (goja.Callable, bool) {
	v := rt.GlobalObject().Get(name)
	if v == nil || goja.IsUndefined(v) {
		return nil, false
	}
	return goja.AssertFunction(v)
}

// Request invokes request(query, params) and converts the returned object.
func (e *JSEngine) Request(ctx context.Context, q engine.Query) (*engine.HTTPRequest, error) {
	rt, err := e.pool.Acquire()
	if err != nil {
		return nil, err
	}
	healthy := true
	defer func() { e.pool.Release(rt, healthy) }()

	fn, ok := callable(rt, "request")
	if !ok {
		healthy = false
		return nil, fmt.Errorf("engine %s: no request() function", e.name)
	}
	params := map[string]any{
		"query": q.Query, "pageno": q.PageNo, "language": q.Locale,
		"safesearch": q.SafeSearch, "time_range": q.TimeRange,
		"headers": map[string]any{}, "cookies": map[string]any{},
		"config": cfgMap(q.Config),
	}
	out, err := fn(goja.Undefined(), rt.ToValue(q.Query), rt.ToValue(params))
	if err != nil {
		healthy = false
		return nil, fmt.Errorf("engine %s request(): %w", e.name, err)
	}
	// A returned object overrides params; otherwise use the (mutated) params.
	var m map[string]any
	if out != nil && !goja.IsUndefined(out) && !goja.IsNull(out) {
		if mm, ok := out.Export().(map[string]any); ok {
			m = mm
		}
	}
	if m == nil {
		m = params
	}
	return mapToRequest(m), nil
}

// Response invokes response(resp) and converts the returned array of results.
func (e *JSEngine) Response(ctx context.Context, resp *engine.HTTPResponse) (result.EngineResults, error) {
	rt, err := e.pool.Acquire()
	if err != nil {
		return nil, err
	}
	healthy := true
	defer func() { e.pool.Release(rt, healthy) }()

	fn, ok := callable(rt, "response")
	if !ok {
		healthy = false
		return nil, fmt.Errorf("engine %s: no response() function", e.name)
	}
	respObj := map[string]any{
		"text": resp.Text(), "url": resp.URL,
		"status_code": resp.StatusCode, "query": resp.Query,
		"config": cfgMap(resp.Config),
	}
	out, err := fn(goja.Undefined(), rt.ToValue(respObj))
	if err != nil {
		healthy = false
		return nil, fmt.Errorf("engine %s response(): %w", e.name, err)
	}
	val := jsruntime.MarshalReply(rt, out)
	rows, ok := val.([]any)
	if !ok {
		return result.EngineResults{}, nil
	}
	res := result.EngineResults{}
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := m["suggestion"].(string); ok && s != "" {
			res = append(res, &result.Suggestion{EngineName: e.name, Value: s})
			continue
		}
		if r := result.FromMap(e.name, m); r != nil {
			res = append(res, r)
		}
	}
	return res, nil
}

// Close drops pooled runtimes.
func (e *JSEngine) Close() { e.pool.Close() }

func cfgMap(c map[string]string) map[string]any {
	m := make(map[string]any, len(c))
	for k, v := range c {
		m[k] = v
	}
	return m
}

func mapToRequest(m map[string]any) *engine.HTTPRequest {
	req := &engine.HTTPRequest{
		Method:  str(m, "method", "GET"),
		URL:     str(m, "url", ""),
		Headers: map[string]string{},
		Cookies: map[string]string{},
	}
	if body := str(m, "data", ""); body != "" {
		req.Body = []byte(body)
	}
	if h, ok := m["headers"].(map[string]any); ok {
		for k, v := range h {
			req.Headers[k] = fmt.Sprintf("%v", v)
		}
	}
	if c, ok := m["cookies"].(map[string]any); ok {
		for k, v := range c {
			req.Cookies[k] = fmt.Sprintf("%v", v)
		}
	}
	return req
}

func str(m map[string]any, key, def string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		if v != nil {
			b, _ := json.Marshal(v)
			return string(b)
		}
	}
	return def
}
