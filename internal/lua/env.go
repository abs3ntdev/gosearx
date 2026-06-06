// Package lua provides the shared, sandboxed gopher-lua environment used by both
// the engine script tier and the plugin tier. Centralizing the sandbox here
// keeps the security boundary in one place: scripts get a curated stdlib
// (url/json/base64) plus the html/xpath API, and nothing else (no os, io,
// require, filesystem, or network).
package lua

import lua "github.com/yuin/gopher-lua"

// NewState returns a fresh sandboxed LState with the curated API installed.
func NewState() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: false})
	Sandbox(L)
	RegisterStdlib(L)
	RegisterHTMLAPI(L)
	return L
}

// Sandbox removes dangerous globals from an LState (filesystem, OS, loaders).
func Sandbox(L *lua.LState) {
	for _, name := range []string{
		"os", "io", "debug", "package",
		"dofile", "loadfile", "load", "loadstring", "require", "collectgarbage",
	} {
		L.SetGlobal(name, lua.LNil)
	}
}
