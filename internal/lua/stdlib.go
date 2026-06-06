package lua

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// RegisterStdlib installs url, json, base64 helper tables (the curated stdlib).
func RegisterStdlib(L *lua.LState) {
	urlMod := L.NewTable()
	L.SetField(urlMod, "encode", L.NewFunction(luaURLEncode))
	L.SetField(urlMod, "escape", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(url.QueryEscape(L.CheckString(1))))
		return 1
	}))
	L.SetField(urlMod, "unescape", L.NewFunction(func(L *lua.LState) int {
		s, err := url.QueryUnescape(L.CheckString(1))
		if err != nil {
			s = L.CheckString(1) // fall back to raw on malformed input
		}
		L.Push(lua.LString(s))
		return 1
	}))
	L.SetGlobal("url", urlMod)

	jsonMod := L.NewTable()
	L.SetField(jsonMod, "decode", L.NewFunction(luaJSONDecode))
	L.SetField(jsonMod, "encode", L.NewFunction(luaJSONEncode))
	L.SetGlobal("json", jsonMod)

	b64Mod := L.NewTable()
	L.SetField(b64Mod, "encode", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(base64.StdEncoding.EncodeToString([]byte(L.CheckString(1)))))
		return 1
	}))
	L.SetField(b64Mod, "decode", L.NewFunction(func(L *lua.LState) int {
		b, err := base64.StdEncoding.DecodeString(L.CheckString(1))
		if err != nil {
			L.RaiseError("base64.decode: %v", err)
			return 0
		}
		L.Push(lua.LString(string(b)))
		return 1
	}))
	L.SetGlobal("base64", b64Mod)
}

func luaURLEncode(L *lua.LState) int {
	v := L.Get(1)
	switch val := v.(type) {
	case lua.LString:
		L.Push(lua.LString(url.QueryEscape(string(val))))
		return 1
	case *lua.LTable:
		values := url.Values{}
		val.ForEach(func(k, vv lua.LValue) { values.Set(k.String(), vv.String()) })
		keys := make([]string, 0, len(values))
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		for i, k := range keys {
			if i > 0 {
				b.WriteByte('&')
			}
			b.WriteString(url.QueryEscape(k))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(values.Get(k)))
		}
		L.Push(lua.LString(b.String()))
		return 1
	default:
		L.ArgError(1, "string or table expected")
		return 0
	}
}

func luaJSONDecode(L *lua.LState) int {
	var data any
	if err := json.Unmarshal([]byte(L.CheckString(1)), &data); err != nil {
		L.RaiseError("json.decode: %v", err)
		return 0
	}
	L.Push(GoToLua(L, data))
	return 1
}

func luaJSONEncode(L *lua.LState) int {
	v := LuaToGo(L.CheckAny(1))
	b, err := json.Marshal(v)
	if err != nil {
		L.RaiseError("json.encode: %v", err)
		return 0
	}
	L.Push(lua.LString(string(b)))
	return 1
}

// GoToLua converts decoded JSON (map/slice/scalar) into Lua values.
func GoToLua(L *lua.LState, v any) lua.LValue {
	switch val := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case []any:
		tbl := L.NewTable()
		for i, e := range val {
			tbl.RawSetInt(i+1, GoToLua(L, e))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, e := range val {
			tbl.RawSetString(k, GoToLua(L, e))
		}
		return tbl
	default:
		return lua.LNil
	}
}

// LuaToGo converts a Lua value into a Go value suitable for json.Marshal.
func LuaToGo(v lua.LValue) any {
	switch val := v.(type) {
	case lua.LBool:
		return bool(val)
	case lua.LNumber:
		return float64(val)
	case lua.LString:
		return string(val)
	case *lua.LTable:
		maxn := val.MaxN()
		if maxn > 0 && maxn == tableLen(val) {
			arr := make([]any, 0, maxn)
			for i := 1; i <= maxn; i++ {
				arr = append(arr, LuaToGo(val.RawGetInt(i)))
			}
			return arr
		}
		obj := map[string]any{}
		val.ForEach(func(k, vv lua.LValue) { obj[k.String()] = LuaToGo(vv) })
		return obj
	default:
		return nil
	}
}

func tableLen(t *lua.LTable) int {
	n := 0
	t.ForEach(func(_, _ lua.LValue) { n++ })
	return n
}
