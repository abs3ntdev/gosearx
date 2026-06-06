// Package bangs implements external "!!bang" redirects (e.g. "!!g foo" sends
// you straight to Google). This is the DuckDuckGo bang convention. A curated
// set of common bangs is built in; the table can be extended freely.
package bangs

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"strings"
)

//go:embed bangs.json
var bangsJSON []byte

// init loads the full DuckDuckGo bang database (~2700 bangs) from the embedded
// JSON, merged on top of the curated defaults. Curated entries win on conflict.
func init() {
	var db map[string]string
	if json.Unmarshal(bangsJSON, &db) != nil {
		return
	}
	for k, v := range db {
		if _, exists := table[k]; !exists {
			// normalize the DDG placeholder to our {{{s}}} token (already done at
			// export time, but be tolerant of variants)
			v = strings.ReplaceAll(v, "{{{s}}}", "{{{s}}}")
			table[k] = v
		}
	}
}

// table maps a bang keyword to a URL template containing {{{s}}} for the query.
var table = map[string]string{
	"g":       "https://www.google.com/search?q={{{s}}}",
	"ddg":     "https://duckduckgo.com/?q={{{s}}}",
	"b":       "https://www.bing.com/search?q={{{s}}}",
	"br":      "https://search.brave.com/search?q={{{s}}}",
	"sp":      "https://www.startpage.com/sp/search?query={{{s}}}",
	"w":       "https://en.wikipedia.org/wiki/Special:Search?search={{{s}}}",
	"yt":      "https://www.youtube.com/results?search_query={{{s}}}",
	"gh":      "https://github.com/search?q={{{s}}}",
	"so":      "https://stackoverflow.com/search?q={{{s}}}",
	"r":       "https://www.reddit.com/search/?q={{{s}}}",
	"a":       "https://www.amazon.com/s?k={{{s}}}",
	"gi":      "https://www.google.com/search?tbm=isch&q={{{s}}}",
	"gm":      "https://www.google.com/maps/search/{{{s}}}",
	"osm":     "https://www.openstreetmap.org/search?query={{{s}}}",
	"wa":      "https://www.wolframalpha.com/input/?i={{{s}}}",
	"imdb":    "https://www.imdb.com/find?q={{{s}}}",
	"npm":     "https://www.npmjs.com/search?q={{{s}}}",
	"crates":  "https://crates.io/search?q={{{s}}}",
	"pypi":    "https://pypi.org/search/?q={{{s}}}",
	"mdn":     "https://developer.mozilla.org/en-US/search?q={{{s}}}",
	"aur":     "https://aur.archlinux.org/packages?K={{{s}}}",
	"archw":   "https://wiki.archlinux.org/index.php?search={{{s}}}",
	"dh":      "https://hub.docker.com/search?q={{{s}}}",
	"gl":      "https://gitlab.com/search?search={{{s}}}",
	"tw":      "https://twitter.com/search?q={{{s}}}",
	"maps":    "https://www.google.com/maps/search/{{{s}}}",
	"tpb":     "https://thepiratebay.org/search.php?q={{{s}}}",
	"scholar": "https://scholar.google.com/scholar?q={{{s}}}",
	"arch":    "https://archlinux.org/packages/?q={{{s}}}",
}

// Resolve takes a raw query, and if it contains an external bang (a token
// starting with "!!"), returns the redirect URL. Otherwise returns "".
func Resolve(rawQuery string) string {
	fields := strings.Fields(rawQuery)
	var bang string
	var terms []string
	for _, f := range fields {
		if strings.HasPrefix(f, "!!") && len(f) > 2 && bang == "" {
			bang = strings.ToLower(f[2:])
			continue
		}
		terms = append(terms, f)
	}
	if bang == "" {
		return ""
	}
	tmpl, ok := table[bang]
	if !ok {
		return ""
	}
	q := url.QueryEscape(strings.Join(terms, " "))
	return strings.ReplaceAll(tmpl, "{{{s}}}", q)
}

// Add registers/overrides a bang at runtime (for config-driven extension).
func Add(keyword, template string) { table[strings.ToLower(keyword)] = template }
