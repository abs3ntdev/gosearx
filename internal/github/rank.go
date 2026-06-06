package github

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/result"
)

// RankedItem is one GitHub result in the unified, relevance-ranked list. Kind
// tells the frontend which card to render; exactly one of the typed pointers is
// set.
type RankedItem struct {
	Kind       string               `json:"kind"`
	Score      float64              `json:"score"`
	Repo       *result.GHRepo       `json:"repo,omitempty"`
	Code       *result.GHCode       `json:"code,omitempty"`
	Issue      *result.GHIssue      `json:"issue,omitempty"`
	User       *result.GHUser       `json:"user,omitempty"`
	Topic      *result.GHTopic      `json:"topic,omitempty"`
	Commit     *result.GHCommit     `json:"commit,omitempty"`
	Discussion *result.GHDiscussion `json:"discussion,omitempty"`
}

// Rank flattens all GitHub result types into a single list ordered by relevance
// to the query, so the best match (whatever its type) surfaces first instead of
// the user scrolling past every repo to reach an issue.
//
// Scoring blends:
//   - textual relevance: how well the item's primary text matches query terms
//     (exact match >> all-terms >> some-terms), the dominant signal
//   - a per-type base weight (repos/issues slightly favored as "primary")
//   - popularity: log(stars) for repos, comments for issues/discussions
//   - recency: a mild boost for recently updated items
func Rank(
	query string,
	repos []*result.GHRepo, code []*result.GHCode, issues []*result.GHIssue,
	users []*result.GHUser, topics []*result.GHTopic, commits []*result.GHCommit,
	discussions []*result.GHDiscussion,
) []RankedItem {
	terms := tokenize(query)
	var items []RankedItem

	for _, r := range repos {
		s := textScore(r.FullName, terms)*1.0 + textScore(r.Description, terms)*0.3
		// Strong boost when the repo's NAME (basename) exactly matches the query
		// — "tern" should rank ternjs/tern above ternjs/tern_for_vim.
		if name := repoBasename(r.FullName); exactMatch(name, terms) {
			s += 2.5
		}
		s += 0.15 // base weight
		s += popularity(float64(r.Stars))
		s += recency(r.Updated) * 0.5
		items = append(items, RankedItem{Kind: "repo", Score: s, Repo: r})
	}
	for _, i := range issues {
		s := textScore(i.Title, terms)*0.95 + textScore(i.Repo, terms)*0.2 + textScore(i.Body, terms)*0.15
		s += float64(i.Comments) * 0.01
		s += recency(i.Created) * 0.4
		items = append(items, RankedItem{Kind: "issue", Score: s, Issue: i})
	}
	for _, d := range discussions {
		s := textScore(d.Title, terms)*0.9 + textScore(d.Repo, terms)*0.2 + textScore(d.Body, terms)*0.15
		s += float64(d.Comments) * 0.01
		if d.Answered {
			s += 0.05
		}
		items = append(items, RankedItem{Kind: "discussion", Score: s, Discussion: d})
	}
	for _, c := range code {
		// match against path + repo; code is supporting, slightly lower base
		s := textScore(c.Path, terms)*0.7 + textScore(c.Repo, terms)*0.4
		items = append(items, RankedItem{Kind: "code", Score: s, Code: c})
	}
	for _, u := range users {
		s := textScore(u.Login, terms)*0.95 + textScore(u.Name, terms)*0.3
		// exact-login match is a very strong signal ("tern" -> user "tern")
		if exactMatch(u.Login, terms) {
			s += 2.5
		}
		items = append(items, RankedItem{Kind: "user", Score: s, User: u})
	}
	for _, c := range commits {
		s := textScore(c.Message, terms)*0.7 + textScore(c.Repo, terms)*0.3
		s += recency(c.Date) * 0.3
		items = append(items, RankedItem{Kind: "commit", Score: s, Commit: c})
	}
	for _, t := range topics {
		s := textScore(t.Name, terms)*0.8 + textScore(t.Description, terms)*0.2
		items = append(items, RankedItem{Kind: "topic", Score: s, Topic: t})
	}

	sort.SliceStable(items, func(a, b int) bool { return items[a].Score > items[b].Score })
	return items
}

// repoBasename returns the repo name without the owner ("a/b" -> "b").
func repoBasename(full string) string {
	if i := strings.LastIndexByte(full, '/'); i >= 0 {
		return full[i+1:]
	}
	return full
}

// exactMatch reports whether text equals the single query term (case-insensitive).
func exactMatch(text string, terms []string) bool {
	return len(terms) == 1 && strings.EqualFold(text, terms[0])
}

func tokenize(q string) []string {
	var out []string
	for _, t := range strings.Fields(strings.ToLower(q)) {
		// drop qualifier tokens like "is:pr", "repo:x" from the text-match terms
		if strings.Contains(t, ":") {
			continue
		}
		t = strings.Trim(t, "#@")
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// textScore rates how well text matches the query terms.
func textScore(text string, terms []string) float64 {
	if text == "" || len(terms) == 0 {
		return 0
	}
	lt := strings.ToLower(text)
	joined := strings.Join(terms, " ")

	// exact match (the whole query equals the text) is the strongest signal
	if lt == joined {
		return 3.0
	}
	// the text is exactly one term (e.g. repo literally named the query)
	if len(terms) == 1 && lt == terms[0] {
		return 3.0
	}

	matched := 0
	var score float64
	for _, term := range terms {
		if strings.Contains(lt, term) {
			matched++
			// whole-word / boundary matches score higher than substring
			if containsWord(lt, term) {
				score += 1.0
			} else {
				score += 0.5
			}
		}
	}
	if matched == 0 {
		return 0
	}
	// bonus for matching ALL terms
	if matched == len(terms) {
		score += 1.0
	}
	// bonus if the text starts with the first term
	if strings.HasPrefix(lt, terms[0]) {
		score += 0.5
	}
	return score
}

func containsWord(text, word string) bool {
	idx := strings.Index(text, word)
	for idx >= 0 {
		before := idx == 0 || !isWordChar(text[idx-1])
		afterPos := idx + len(word)
		after := afterPos >= len(text) || !isWordChar(text[afterPos])
		if before && after {
			return true
		}
		next := strings.Index(text[idx+1:], word)
		if next < 0 {
			break
		}
		idx = idx + 1 + next
	}
	return false
}

func isWordChar(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}

// popularity returns a damped log boost from a count (stars/forks).
func popularity(n float64) float64 {
	if n <= 0 {
		return 0
	}
	return math.Log10(n+1) * 0.3
}

// recency returns a 0..1 boost for items updated recently (decays over ~2 years).
func recency(ts string) float64 {
	if ts == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0
	}
	days := time.Since(t).Hours() / 24
	if days < 0 {
		days = 0
	}
	// 1.0 today -> ~0 at 730 days
	v := 1.0 - days/730.0
	if v < 0 {
		return 0
	}
	return v
}
