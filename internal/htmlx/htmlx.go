// Package htmlx provides the small set of HTML/XPath helpers that mirror the
// lxml + searx/utils functions SearXNG engines rely on (extract_text,
// eval_xpath, eval_xpath_list, extract_url). Both the Lua bindings and the
// declarative YAML engine build on this package so behavior is identical across
// tiers.
package htmlx

import (
	"net/url"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// Parse parses an HTML document and returns the root node.
func Parse(src string) (*html.Node, error) {
	return htmlquery.Parse(strings.NewReader(src))
}

// List evaluates an XPath expression returning all matching nodes
// (eval_xpath_list). A nil base node yields an empty slice.
func List(node *html.Node, expr string) ([]*html.Node, error) {
	if node == nil {
		return nil, nil
	}
	return htmlquery.QueryAll(node, expr)
}

// First evaluates an XPath expression returning the first match or nil
// (eval_xpath).
func First(node *html.Node, expr string) (*html.Node, error) {
	if node == nil {
		return nil, nil
	}
	return htmlquery.Query(node, expr)
}

// Text returns the normalized text of the first match (extract_text). Empty
// string if no match.
func Text(node *html.Node, expr string) (string, error) {
	n, err := First(node, expr)
	if err != nil || n == nil {
		return "", err
	}
	return NormalizeSpace(NodeText(n)), nil
}

// URL returns the first match's text resolved against base into an absolute URL
// (extract_url).
func URL(node *html.Node, expr, base string) (string, error) {
	n, err := First(node, expr)
	if err != nil || n == nil {
		return "", err
	}
	return ResolveURL(base, strings.TrimSpace(NodeText(n))), nil
}

// Attr returns the named attribute of the first match.
func Attr(node *html.Node, expr, attr string) (string, error) {
	n, err := First(node, expr)
	if err != nil || n == nil {
		return "", err
	}
	return htmlquery.SelectAttr(n, attr), nil
}

// NodeText returns the text payload of a node: inner text for elements,
// raw data for attribute/text nodes.
func NodeText(n *html.Node) string {
	switch n.Type {
	case html.TextNode:
		return n.Data
	case html.ElementNode, html.DocumentNode:
		return htmlquery.InnerText(n)
	default:
		if n.Data != "" {
			return n.Data
		}
		return htmlquery.InnerText(n)
	}
}

// ResolveURL resolves ref against base, returning an absolute URL.
func ResolveURL(base, ref string) string {
	if ref == "" {
		return ""
	}
	if base == "" {
		return ref
	}
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

// NormalizeSpace collapses whitespace runs and trims (mirrors extract_text).
func NormalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// StripHTML removes any HTML markup from a string and returns normalized plain
// text (used for titles, which must be plain).
func StripHTML(s string) string {
	if s == "" {
		return ""
	}
	if !strings.ContainsAny(s, "<&") {
		return NormalizeSpace(s)
	}
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	return NormalizeSpace(htmlquery.InnerText(doc))
}

// allowed tags kept by SanitizeHTML (a safe formatting subset, like SearXNG).
var allowedTags = map[string]bool{
	"b": true, "strong": true, "i": true, "em": true, "u": true,
	"br": true, "p": true, "span": true, "small": true, "sub": true,
	"sup": true, "mark": true, "code": true, "img": true,
}

// allowed attributes per tag.
var allowedAttrs = map[string]map[string]bool{
	"img": {"src": true, "alt": true, "width": true, "height": true},
}

// SanitizeHTML returns result content with only a safe subset of formatting
// tags preserved (bold/italic/line-breaks and valid images). Disallowed tags
// are unwrapped (their text kept), <script>/<style> are dropped entirely, and
// <img> with an empty/relative-junk src is removed so broken favicon stubs from
// upstream engines don't render. This is the rendered-HTML successor to
// SearXNG's content sanitization.
func SanitizeHTML(s string) string {
	if s == "" || !strings.Contains(s, "<") {
		return NormalizeSpace(s)
	}
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return StripHTML(s)
	}
	var b strings.Builder
	// Find the body and render its sanitized children.
	var body *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}
		for c := n.FirstChild; c != nil && body == nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	if body == nil {
		return StripHTML(s)
	}
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		sanitizeNode(&b, c)
	}
	return strings.TrimSpace(collapseSpaces(b.String()))
}

func sanitizeNode(b *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		b.WriteString(html.EscapeString(n.Data))
		return
	case html.ElementNode:
		// drop dangerous containers entirely (including their content)
		if n.Data == "script" || n.Data == "style" {
			return
		}
		if !allowedTags[n.Data] {
			// unwrap: keep children, drop the tag
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				sanitizeNode(b, c)
			}
			return
		}
		if n.Data == "img" {
			src := getAttr(n, "src")
			// drop broken/empty/non-http images (upstream favicon stubs)
			if src == "" || !(strings.HasPrefix(src, "http") || strings.HasPrefix(src, "//")) {
				return
			}
		}
		b.WriteByte('<')
		b.WriteString(n.Data)
		for _, a := range n.Attr {
			if allowedAttrs[n.Data] != nil && allowedAttrs[n.Data][a.Key] {
				b.WriteByte(' ')
				b.WriteString(a.Key)
				b.WriteString(`="`)
				b.WriteString(html.EscapeString(a.Val))
				b.WriteByte('"')
			}
		}
		if n.Data == "br" || n.Data == "img" {
			b.WriteString("/>")
			return
		}
		b.WriteByte('>')
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sanitizeNode(b, c)
		}
		b.WriteString("</")
		b.WriteString(n.Data)
		b.WriteByte('>')
	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sanitizeNode(b, c)
		}
	}
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func collapseSpaces(s string) string {
	// collapse runs of spaces/tabs/newlines to single spaces, but keep <br>.
	var b strings.Builder
	space := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}
