// container.go ports SearXNG's ResultContainer logic (searx/results.py):
// collecting results from many engines, deduplicating + merging main results,
// scoring them, and producing a final ordered, grouped list. Answers and
// suggestions are collected separately.
package result

import (
	"net/url"
	"sort"
	"strings"
)

// Container accumulates results from all engines for one search.
type Container struct {
	mainByHash  map[string]*MainResult
	mainOrder   []*MainResult
	suggestions map[string]*Suggestion
	corrections map[string]*Correction
	answers     []*Answer
	infoboxes   []*Infobox
	images      []*Image
	videos      []*Video
	papers      []*Paper
	torrents    []*Torrent
	maps        []*MapResult
	codes       []*Code
	files       []*File
	keyvalues   []*KeyValue
	quotes      []*Quote
	charts      []*Chart
	ghRepos     []*GHRepo
	ghCode      []*GHCode
	ghIssues    []*GHIssue
	ghUsers     []*GHUser
	ghTopics    []*GHTopic
	ghCommits   []*GHCommit
	ghDiscuss   []*GHDiscussion

	// engineWeights lets scoring weight engines (default 1.0).
	engineWeights map[string]float64

	// domainPenalties down-ranks low-quality domains (nil = use defaults).
	domainPenalties map[string]float64
	// collapseDups enables near-duplicate (same host + title) folding.
	collapseDups bool

	// onResult, if set, is called per main result; returning false drops it.
	// This is the integration point for plugin on_result hooks.
	onResult func(*MainResult) bool
}

// SetCollapseDuplicates toggles near-duplicate collapsing in Ordered().
func (c *Container) SetCollapseDuplicates(on bool) { c.collapseDups = on }

// NewContainer returns an empty result container.
func NewContainer() *Container {
	return &Container{
		mainByHash:    map[string]*MainResult{},
		suggestions:   map[string]*Suggestion{},
		corrections:   map[string]*Correction{},
		engineWeights: map[string]float64{},
	}
}

// SetOnResult registers a per-result filter (plugin on_result integration).
func (c *Container) SetOnResult(fn func(*MainResult) bool) { c.onResult = fn }

// SetEngineWeight registers a scoring weight for an engine.
func (c *Container) SetEngineWeight(engine string, w float64) {
	c.engineWeights[engine] = w
}

// AddFromEngine ingests an engine's results. position is the 1-based rank of
// each result within that engine's response (used for scoring).
func (c *Container) AddFromEngine(results EngineResults) {
	position := 0
	for _, r := range results {
		switch v := r.(type) {
		case *MainResult:
			if c.onResult != nil && !c.onResult(v) {
				continue // dropped by a plugin
			}
			position++
			c.mergeMain(v, position)
		case *Suggestion:
			if _, ok := c.suggestions[v.Value]; !ok {
				c.suggestions[v.Value] = v
			}
		case *Answer:
			c.answers = append(c.answers, v)
		case *Correction:
			if _, ok := c.corrections[v.Value]; !ok {
				c.corrections[v.Value] = v
			}
		case *Infobox:
			c.infoboxes = append(c.infoboxes, v)
		case *Image:
			c.images = append(c.images, v)
		case *Video:
			c.videos = append(c.videos, v)
		case *Paper:
			c.papers = append(c.papers, v)
		case *Torrent:
			c.torrents = append(c.torrents, v)
		case *MapResult:
			c.maps = append(c.maps, v)
		case *Code:
			c.codes = append(c.codes, v)
		case *File:
			c.files = append(c.files, v)
		case *KeyValue:
			c.keyvalues = append(c.keyvalues, v)
		case *Quote:
			c.quotes = append(c.quotes, v)
		case *Chart:
			c.charts = append(c.charts, v)
		case *GHRepo:
			c.ghRepos = append(c.ghRepos, v)
		case *GHCode:
			c.ghCode = append(c.ghCode, v)
		case *GHIssue:
			c.ghIssues = append(c.ghIssues, v)
		case *GHUser:
			c.ghUsers = append(c.ghUsers, v)
		case *GHTopic:
			c.ghTopics = append(c.ghTopics, v)
		case *GHCommit:
			c.ghCommits = append(c.ghCommits, v)
		case *GHDiscussion:
			c.ghDiscuss = append(c.ghDiscuss, v)
		}
	}
}

// Answers returns collected instant answers.
func (c *Container) Answers() []*Answer { return c.answers }

// Infoboxes returns collected infoboxes.
func (c *Container) Infoboxes() []*Infobox { return c.infoboxes }

// Corrections returns collected "did you mean" corrections, sorted.
func (c *Container) Corrections() []string {
	out := make([]string, 0, len(c.corrections))
	for s := range c.corrections {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// Images returns collected image results.
func (c *Container) Images() []*Image { return c.images }

// Videos returns collected video results.
func (c *Container) Videos() []*Video { return c.videos }

// Rich template result accessors.
func (c *Container) Papers() []*Paper       { return c.papers }
func (c *Container) Torrents() []*Torrent   { return c.torrents }
func (c *Container) Maps() []*MapResult     { return c.maps }
func (c *Container) Codes() []*Code         { return c.codes }
func (c *Container) Files() []*File         { return c.files }
func (c *Container) KeyValues() []*KeyValue { return c.keyvalues }

// Quotes returns collected finance quotes.
func (c *Container) Quotes() []*Quote { return c.quotes }

// Charts returns collected finance charts.
func (c *Container) Charts() []*Chart { return c.charts }

// GitHub result accessors.
func (c *Container) GHRepos() []*GHRepo             { return c.ghRepos }
func (c *Container) GHCode() []*GHCode              { return c.ghCode }
func (c *Container) GHIssues() []*GHIssue           { return c.ghIssues }
func (c *Container) GHUsers() []*GHUser             { return c.ghUsers }
func (c *Container) GHTopics() []*GHTopic           { return c.ghTopics }
func (c *Container) GHCommits() []*GHCommit         { return c.ghCommits }
func (c *Container) GHDiscussions() []*GHDiscussion { return c.ghDiscuss }

// mergeMain deduplicates by content hash and merges duplicates, accumulating
// positions and engine names (port of _merge_main_result / merge_two_main_results).
func (c *Container) mergeMain(r *MainResult, position int) {
	h := mainHash(r)
	existing, ok := c.mainByHash[h]
	if !ok {
		r.Positions = []int{position}
		if len(r.Engines) == 0 && r.EngineName != "" {
			r.Engines = []string{r.EngineName}
		}
		c.mainByHash[h] = r
		c.mainOrder = append(c.mainOrder, r)
		return
	}
	// Merge into existing: accumulate position, union engines, prefer richer text.
	existing.Positions = append(existing.Positions, position)
	existing.Engines = addUnique(existing.Engines, r.EngineName)
	if len(r.Content) > len(existing.Content) {
		existing.Content = r.Content
	}
	if len(r.Title) > len(existing.Title) {
		existing.Title = r.Title
	}
	// Prefer https over http (SearXNG behavior).
	if strings.HasPrefix(r.URL, "https://") && strings.HasPrefix(existing.URL, "http://") {
		existing.URL = r.URL
	}
	if existing.Thumbnail == "" && r.Thumbnail != "" {
		existing.Thumbnail = r.Thumbnail
	}
}

// Ordered finalizes scores and returns results sorted by score (desc) then
// grouped so same-type results cluster (port of get_ordered_results).
func (c *Container) Ordered() []*MainResult {
	for _, r := range c.mainOrder {
		r.Score = c.score(r)
	}
	out := make([]*MainResult, len(c.mainOrder))
	copy(out, c.mainOrder)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if c.collapseDups {
		out = collapseNearDuplicates(out)
	}
	return group(out)
}

// Suggestions returns the collected suggestions, sorted for determinism.
func (c *Container) Suggestions() []string {
	out := make([]string, 0, len(c.suggestions))
	for s := range c.suggestions {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// score ports calculate_score: weight = product(engine weights) * len(positions),
// then score += weight/position for each position (normal priority).
func (c *Container) score(r *MainResult) float64 {
	weight := 1.0
	for _, e := range r.Engines {
		if w, ok := c.engineWeights[e]; ok {
			weight *= w
		}
	}
	weight *= float64(len(r.Positions))
	var score float64
	for _, p := range r.Positions {
		if p > 0 {
			score += weight / float64(p)
		}
	}
	// Apply SEO / content-farm down-ranking.
	return score * c.domainPenalty(r)
}

// mainHash is the dedup key (port of MainResult.__hash__): dedup by location
// (netloc|path|query|fragment) so the SAME page from different engines merges
// into one result — and the merge can then keep a thumbnail contributed by any
// engine (e.g. Brave). Thumbnail is deliberately NOT part of the key.
func mainHash(r *MainResult) string {
	u, err := url.Parse(r.URL)
	if err != nil {
		return string(r.Kind()) + "|" + r.URL
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	path := strings.TrimSuffix(u.Path, "/")
	return strings.Join([]string{
		string(r.Kind()), host, path, u.RawQuery, u.Fragment,
	}, "|")
}

// group clusters results of the same category/type within a sliding window so
// similar results display together (simplified port of the grouping pass).
func group(in []*MainResult) []*MainResult {
	const maxCount = 8
	const maxDistance = 20
	out := make([]*MainResult, 0, len(in))
	used := make([]bool, len(in))
	for i := range in {
		if used[i] {
			continue
		}
		out = append(out, in[i])
		used[i] = true
		count := 0
		for j := i + 1; j < len(in) && j <= i+maxDistance && count < maxCount; j++ {
			if used[j] {
				continue
			}
			if in[j].Type == in[i].Type {
				out = append(out, in[j])
				used[j] = true
				count++
			}
		}
	}
	return out
}

func addUnique(s []string, v string) []string {
	if v == "" {
		return s
	}
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
