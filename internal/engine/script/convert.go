// convert.go bridges Lua tables <-> typed Go structs for the engine boundary.
package script

import (
	"github.com/searxng/gosearx/internal/engine"
	golua "github.com/searxng/gosearx/internal/lua"
	"github.com/searxng/gosearx/internal/result"
	lua "github.com/yuin/gopher-lua"
)

// tableToRequest converts the `params` table a script returns into an HTTPRequest.
func tableToRequest(t *lua.LTable) *engine.HTTPRequest {
	req := &engine.HTTPRequest{
		Method:  optString(t, "method", "GET"),
		URL:     optString(t, "url", ""),
		Headers: map[string]string{},
		Cookies: map[string]string{},
	}
	if body := optString(t, "data", ""); body != "" {
		req.Body = []byte(body)
	}
	if h, ok := t.RawGetString("headers").(*lua.LTable); ok {
		h.ForEach(func(k, v lua.LValue) { req.Headers[k.String()] = v.String() })
	}
	if c, ok := t.RawGetString("cookies").(*lua.LTable); ok {
		c.ForEach(func(k, v lua.LValue) { req.Cookies[k.String()] = v.String() })
	}
	return req
}

// tableToResults converts the array of result tables a script returns into
// typed result.Result values. A table with a "suggestion" key becomes a
// Suggestion; otherwise it is a MainResult (mirrors SearXNG's dict dispatch).
// tableToResults converts the array of result tables a script returns into
// typed results. Each row is dispatched via result.FromMap on its "type" key
// (or special keys like suggestion/answer/correction), so engines can emit
// main results, suggestions, answers, infoboxes, images, quotes, and charts.
func tableToResults(engineName string, t *lua.LTable) result.EngineResults {
	out := result.EngineResults{}
	t.ForEach(func(_, v lua.LValue) {
		row, ok := v.(*lua.LTable)
		if !ok {
			return
		}
		// suggestion is the one legacy key not carrying a "type".
		if s := optString(row, "suggestion", ""); s != "" {
			out = append(out, &result.Suggestion{EngineName: engineName, Value: s})
			return
		}
		m, _ := golua.LuaToGo(row).(map[string]any)
		if m == nil {
			return
		}
		if r := result.FromMap(engineName, m); r != nil {
			out = append(out, r)
		}
	})
	return out
}

func optString(t *lua.LTable, key, def string) string {
	if v, ok := t.RawGetString(key).(lua.LString); ok {
		return string(v)
	}
	return def
}
