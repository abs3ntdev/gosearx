// enrich.go adds result-quality passes on top of the SearXNG-derived container:
//   - SEO / content-farm down-ranking via a configurable per-domain penalty
//   - near-duplicate collapsing (same host + near-identical normalized title)
//
// Both are opt-in refinements applied during Ordered(); they never drop a
// unique page, they only re-rank and fold obvious near-dupes.
package result

import (
	"net/url"
	"strings"
)

// defaultDomainPenalties down-weights low-quality / content-farm domains. The
// value multiplies the result's score (1.0 = no change, 0.5 = halve). Operators
// can extend/override this via SetDomainPenalties.
//
// Kept intentionally small and uncontroversial; tune per instance.
var defaultDomainPenalties = map[string]float64{
	"pinterest.com":     0.4,
	"quora.com":         0.6,
	"w3schools.com":     0.7,
	"geeksforgeeks.org": 0.7,
	"answers.com":       0.4,
	"ehow.com":          0.4,
	"wikihow.com":       0.7,
	"fandom.com":        0.7,
	"slideshare.net":    0.6,
	"scribd.com":        0.4,
	"coursehero.com":    0.4,
}

// domainPenalty returns the score multiplier for a result's host (1.0 if none).
// Matches the registered base domain or any subdomain of it.
func (c *Container) domainPenalty(r *MainResult) float64 {
	penalties := c.domainPenalties
	if penalties == nil {
		penalties = defaultDomainPenalties
	}
	if len(penalties) == 0 {
		return 1.0
	}
	u, err := url.Parse(r.URL)
	if err != nil {
		return 1.0
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if p, ok := penalties[host]; ok {
		return p
	}
	// subdomain match: foo.pinterest.com -> pinterest.com
	for dom, p := range penalties {
		if strings.HasSuffix(host, "."+dom) {
			return p
		}
	}
	return 1.0
}

// SetDomainPenalties overrides the default SEO penalty table. Passing nil
// restores defaults; passing an empty map disables penalties entirely.
func (c *Container) SetDomainPenalties(m map[string]float64) {
	if m == nil {
		c.domainPenalties = nil
		return
	}
	c.domainPenalties = m
}

// collapseNearDuplicates folds results that point at effectively the same page:
// same host and a near-identical normalized title. The highest-scored survivor
// keeps the others' engines/positions so its score reflects the consensus.
//
// This catches syndicated/mirrored content that escapes URL-based dedup (e.g.
// "example.com/article" vs "example.com/article?ref=rss" already merge, but
// "news-a.com/x" and "news-b.com/x" do not — those are left alone since the host
// differs; we only collapse within a host to stay conservative).
func collapseNearDuplicates(in []*MainResult) []*MainResult {
	type key struct{ host, title string }
	seen := map[key]*MainResult{}
	out := make([]*MainResult, 0, len(in))
	for _, r := range in {
		k := key{host: hostOf(r.URL), title: normalizeTitle(r.Title)}
		if k.title == "" || k.host == "" {
			out = append(out, r)
			continue
		}
		if prev, ok := seen[k]; ok {
			// Fold this duplicate into the earlier (higher-ranked) survivor.
			prev.Positions = append(prev.Positions, r.Positions...)
			for _, e := range r.Engines {
				prev.Engines = addUnique(prev.Engines, e)
			}
			if len(r.Content) > len(prev.Content) {
				prev.Content = r.Content
			}
			continue
		}
		seen[k] = r
		out = append(out, r)
	}
	return out
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Host), "www.")
}

// normalizeTitle lowercases, strips trailing site-name suffixes after a
// separator, and collapses whitespace/punctuation so cosmetic differences
// don't defeat near-dup detection.
func normalizeTitle(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	// Drop a trailing " - Site Name" / " | Site Name" tail (common SEO pattern).
	for _, sep := range []string{" | ", " - ", " — ", " · ", " :: "} {
		if i := strings.LastIndex(t, sep); i > 10 {
			t = t[:i]
		}
	}
	var b strings.Builder
	lastSpace := false
	for _, r := range t {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSpace = false
		default:
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}
