package lua

import (
	"github.com/antchfx/htmlquery"
	"github.com/searxng/gosearx/internal/htmlx"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/net/html"
)

const luaNodeType = "html.node"

// RegisterHTMLAPI installs the html and xpath tables (over internal/htmlx).
func RegisterHTMLAPI(L *lua.LState) {
	mt := L.NewTypeMetatable(luaNodeType)
	L.SetField(mt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		n := checkNode(L, 1)
		L.Push(lua.LString(htmlx.NormalizeSpace(htmlquery.InnerText(n))))
		return 1
	}))

	htmlMod := L.NewTable()
	L.SetField(htmlMod, "parse", L.NewFunction(luaHTMLParse))
	L.SetGlobal("html", htmlMod)

	xpathMod := L.NewTable()
	L.SetField(xpathMod, "list", L.NewFunction(luaXPathList))
	L.SetField(xpathMod, "first", L.NewFunction(luaXPathFirst))
	L.SetField(xpathMod, "text", L.NewFunction(luaXPathText))
	L.SetField(xpathMod, "attr", L.NewFunction(luaXPathAttr))
	L.SetField(xpathMod, "url", L.NewFunction(luaXPathURL))
	L.SetGlobal("xpath", xpathMod)
}

func pushNode(L *lua.LState, n *html.Node) {
	ud := L.NewUserData()
	ud.Value = n
	L.SetMetatable(ud, L.GetTypeMetatable(luaNodeType))
	L.Push(ud)
}

func checkNode(L *lua.LState, idx int) *html.Node {
	ud := L.CheckUserData(idx)
	if n, ok := ud.Value.(*html.Node); ok {
		return n
	}
	L.ArgError(idx, "html node expected")
	return nil
}

func optNode(L *lua.LState, idx int) *html.Node {
	v := L.Get(idx)
	if v == lua.LNil {
		return nil
	}
	ud, ok := v.(*lua.LUserData)
	if !ok {
		L.ArgError(idx, "html node or nil expected")
		return nil
	}
	n, _ := ud.Value.(*html.Node)
	return n
}

func luaHTMLParse(L *lua.LState) int {
	doc, err := htmlx.Parse(L.CheckString(1))
	if err != nil {
		L.RaiseError("html.parse: %v", err)
		return 0
	}
	pushNode(L, doc)
	return 1
}

func luaXPathList(L *lua.LState) int {
	node := optNode(L, 1)
	expr := L.CheckString(2)
	tbl := L.NewTable()
	nodes, err := htmlx.List(node, expr)
	if err != nil {
		L.RaiseError("xpath.list(%q): %v", expr, err)
		return 0
	}
	for i, n := range nodes {
		pushNode(L, n)
		tbl.RawSetInt(i+1, L.Get(-1))
		L.Pop(1)
	}
	L.Push(tbl)
	return 1
}

func luaXPathFirst(L *lua.LState) int {
	node := optNode(L, 1)
	expr := L.CheckString(2)
	n, err := htmlx.First(node, expr)
	if err != nil {
		L.RaiseError("xpath.first(%q): %v", expr, err)
		return 0
	}
	if n == nil {
		L.Push(lua.LNil)
		return 1
	}
	pushNode(L, n)
	return 1
}

func luaXPathText(L *lua.LState) int {
	node := optNode(L, 1)
	expr := L.CheckString(2)
	s, err := htmlx.Text(node, expr)
	if err != nil {
		L.RaiseError("xpath.text(%q): %v", expr, err)
		return 0
	}
	L.Push(lua.LString(s))
	return 1
}

func luaXPathAttr(L *lua.LState) int {
	node := optNode(L, 1)
	expr := L.CheckString(2)
	attr := L.CheckString(3)
	s, err := htmlx.Attr(node, expr, attr)
	if err != nil {
		L.RaiseError("xpath.attr(%q): %v", expr, err)
		return 0
	}
	L.Push(lua.LString(s))
	return 1
}

func luaXPathURL(L *lua.LState) int {
	node := optNode(L, 1)
	expr := L.CheckString(2)
	base := L.OptString(3, "")
	s, err := htmlx.URL(node, expr, base)
	if err != nil {
		L.RaiseError("xpath.url(%q): %v", expr, err)
		return 0
	}
	L.Push(lua.LString(s))
	return 1
}
