package lua

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// engineHTTPClient is a shared, bounded client for engine follow-up requests
// (e.g. fetching a detail/summary endpoint, or a token + search two-step). A
// cookie jar is attached so multi-step flows (Startpage's sc handshake) keep
// session cookies. Only engines get this capability — plugins stay network-free.
var engineHTTPClient = func() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 8 * time.Second, Jar: jar}
}()

// RegisterHTTPAPI installs a constrained `http` table with a single `get`
// function. This is the controlled network capability for engine scripts; it
// mirrors what SearXNG engines do via searx.network.get for multi-request
// engines (e.g. Wikipedia's opensearch list + REST summary).
//
//	resp = http.get(url[, headers_table])
//	-> { status_code = 200, text = "...", url = "..." } or nil on error
func RegisterHTTPAPI(L *lua.LState) {
	mod := L.NewTable()
	L.SetField(mod, "get", L.NewFunction(luaHTTPGet))
	L.SetField(mod, "post", L.NewFunction(luaHTTPPost))
	L.SetGlobal("http", mod)
}

// luaHTTPPost implements http.post(url, body[, headers]) for engines that need
// a follow-up form POST (e.g. Startpage's sc-token search).
//
//	resp = http.post(url, "a=1&b=2", { ["Content-Type"]="application/x-www-form-urlencoded" })
func luaHTTPPost(L *lua.LState) int {
	target := L.CheckString(1)
	body := L.OptString(2, "")
	req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(body))
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (gosearx engine)")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if h, ok := L.Get(3).(*lua.LTable); ok {
		h.ForEach(func(k, v lua.LValue) { req.Header.Set(k.String(), v.String()) })
	}
	if ctx := L.Context(); ctx != nil {
		req = req.WithContext(ctx)
	}
	resp, err := engineHTTPClient.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	out := L.NewTable()
	out.RawSetString("status_code", lua.LNumber(resp.StatusCode))
	out.RawSetString("text", lua.LString(string(data)))
	out.RawSetString("url", lua.LString(resp.Request.URL.String()))
	L.Push(out)
	return 1
}

func luaHTTPGet(L *lua.LState) int {
	target := L.CheckString(1)
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (gosearx engine)")
	req.Header.Set("Accept", "application/json")
	// Optional headers table as arg 2.
	if h, ok := L.Get(2).(*lua.LTable); ok {
		h.ForEach(func(k, v lua.LValue) {
			req.Header.Set(k.String(), v.String())
		})
	}
	// Honor the LState's context deadline if one is set.
	if ctx := L.Context(); ctx != nil {
		req = req.WithContext(ctx)
	}

	resp, err := engineHTTPClient.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	out := L.NewTable()
	out.RawSetString("status_code", lua.LNumber(resp.StatusCode))
	out.RawSetString("text", lua.LString(string(body)))
	out.RawSetString("url", lua.LString(resp.Request.URL.String()))
	L.Push(out)
	return 1
}
