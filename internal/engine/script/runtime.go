// Package script implements the Lua tier of the engine runtime using
// gopher-lua (pure Go, no CGO). It builds on internal/lua for the shared
// sandboxed environment and LState pool, so engines and plugins share one
// security boundary and one concurrency model.
package script

import (
	"context"
	"fmt"

	"github.com/searxng/gosearx/internal/engine"
	golua "github.com/searxng/gosearx/internal/lua"
	"github.com/searxng/gosearx/internal/result"
	lua "github.com/yuin/gopher-lua"
)

// LuaEngine is an engine.Engine backed by a Lua script defining request() and
// response() functions.
type LuaEngine struct {
	name string
	pool *golua.Pool
}

// Compile parses a Lua engine script once and returns a ready LuaEngine.
// Engine states get the controlled http.get capability (for multi-request
// engines like Wikipedia: list + summary).
func Compile(name, src string) (*LuaEngine, error) {
	pool, err := golua.CompileWithHTTP(name, src)
	if err != nil {
		return nil, err
	}
	return &LuaEngine{name: name, pool: pool}, nil
}

func (e *LuaEngine) Name() string { return e.name }

// Request invokes the script's request(query, params) function.
func (e *LuaEngine) Request(ctx context.Context, q engine.Query) (*engine.HTTPRequest, error) {
	L, err := e.pool.Acquire()
	if err != nil {
		return nil, err
	}
	L.SetContext(ctx)
	healthy := true
	defer func() { e.pool.Release(L, healthy) }()

	fn := L.GetGlobal("request")
	if fn == lua.LNil {
		healthy = false
		return nil, fmt.Errorf("engine %s: no request() function", e.name)
	}

	params := L.NewTable()
	params.RawSetString("query", lua.LString(q.Query))
	params.RawSetString("pageno", lua.LNumber(q.PageNo))
	params.RawSetString("language", lua.LString(q.Locale))
	params.RawSetString("safesearch", lua.LNumber(q.SafeSearch))
	params.RawSetString("time_range", lua.LString(q.TimeRange))
	params.RawSetString("headers", L.NewTable())
	params.RawSetString("cookies", L.NewTable())
	cfg := L.NewTable()
	for k, v := range q.Config {
		cfg.RawSetString(k, lua.LString(v))
	}
	params.RawSetString("config", cfg)

	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true},
		lua.LString(q.Query), params); err != nil {
		healthy = false
		return nil, fmt.Errorf("engine %s request(): %w", e.name, err)
	}

	ret := L.Get(-1)
	L.Pop(1)
	out := params
	if t, ok := ret.(*lua.LTable); ok {
		out = t
	}
	return tableToRequest(out), nil
}

// Response invokes the script's response(resp) function.
func (e *LuaEngine) Response(ctx context.Context, resp *engine.HTTPResponse) (result.EngineResults, error) {
	L, err := e.pool.Acquire()
	if err != nil {
		return nil, err
	}
	L.SetContext(ctx)
	healthy := true
	defer func() { e.pool.Release(L, healthy) }()

	fn := L.GetGlobal("response")
	if fn == lua.LNil {
		healthy = false
		return nil, fmt.Errorf("engine %s: no response() function", e.name)
	}

	respTbl := L.NewTable()
	respTbl.RawSetString("text", lua.LString(resp.Text()))
	respTbl.RawSetString("url", lua.LString(resp.URL))
	respTbl.RawSetString("status_code", lua.LNumber(resp.StatusCode))
	respTbl.RawSetString("query", lua.LString(resp.Query))
	rcfg := L.NewTable()
	for k, v := range resp.Config {
		rcfg.RawSetString(k, lua.LString(v))
	}
	respTbl.RawSetString("config", rcfg)

	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, respTbl); err != nil {
		healthy = false
		return nil, fmt.Errorf("engine %s response(): %w", e.name, err)
	}

	ret := L.Get(-1)
	L.Pop(1)
	tbl, ok := ret.(*lua.LTable)
	if !ok {
		return result.EngineResults{}, nil
	}
	return tableToResults(e.name, tbl), nil
}

// Close releases pooled states.
func (e *LuaEngine) Close() { e.pool.Close() }
