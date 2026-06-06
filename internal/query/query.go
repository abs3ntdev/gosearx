// Package query parses a raw user query string into search terms plus engine
// and category selectors. It is a Phase 1 subset of searx/query.py supporting:
//
//	!shortcut   select a specific engine by its shortcut  (e.g. !mj golang)
//	:category   select all engines in a category          (e.g. :general cats)
//
// Selectors may appear anywhere; remaining tokens form the search text.
package query

import "strings"

// Parsed is the result of parsing a raw query.
type Parsed struct {
	Text         string
	EngineShorts []string // explicit !shortcuts
	Categories   []string // :categories
}

// Parse splits a raw query into text + selectors.
func Parse(raw string) Parsed {
	var p Parsed
	var terms []string
	for _, tok := range strings.Fields(raw) {
		switch {
		case strings.HasPrefix(tok, "!") && len(tok) > 1:
			p.EngineShorts = append(p.EngineShorts, tok[1:])
		case strings.HasPrefix(tok, ":") && len(tok) > 1:
			p.Categories = append(p.Categories, tok[1:])
		default:
			terms = append(terms, tok)
		}
	}
	p.Text = strings.Join(terms, " ")
	return p
}
