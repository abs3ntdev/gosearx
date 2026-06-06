package media

import "strings"

// Intent is the parsed media-lookup intent from a query.
type Intent struct {
	Title     string // the cleaned title to search TMDB for
	MediaType string // "", "movie", or "tv" — a hint to bias the search
}

// movieKeywords signal a film lookup.
var movieKeywords = map[string]bool{
	"movie": true, "film": true, "cinema": true,
}

// tvKeywords signal a TV/series lookup.
var tvKeywords = map[string]bool{
	"tv": true, "show": true, "series": true, "season": true, "episode": true,
	"sitcom": true,
}

// genericMediaKeywords trigger media intent without biasing movie vs tv. These
// are the "what Google shows a panel for" signals: trailer, cast, ratings, etc.
var genericMediaKeywords = map[string]bool{
	"trailer": true, "cast": true, "imdb": true, "rottentomatoes": true,
	"streaming": true, "watch": true, "showtimes": true, "plot": true,
	"rating": true, "ratings": true, "review": true, "reviews": true,
}

// multiwordTriggers are phrases (checked on the lowercased full query) that also
// signal media intent, e.g. "where to watch", "rotten tomatoes".
var multiwordTriggers = []string{
	"where to watch", "rotten tomatoes", "tv show", "tv series",
	"how to watch", "cast of",
}

// Detect inspects a query and returns a media Intent if it looks like a
// movie/TV lookup, plus ok=false when it isn't. The title is the query with the
// trigger keywords stripped, so "dune movie" -> title "dune".
func Detect(query string) (Intent, bool) {
	q := strings.TrimSpace(query)
	if q == "" {
		return Intent{}, false
	}
	lower := strings.ToLower(q)

	mediaType := ""
	triggered := false

	// Multi-word phrase triggers (checked first; they also imply a type).
	for _, phrase := range multiwordTriggers {
		if strings.Contains(lower, phrase) {
			triggered = true
			if phrase == "tv show" || phrase == "tv series" {
				mediaType = "tv"
			}
			lower = strings.ReplaceAll(lower, phrase, " ")
		}
	}

	// Single-word keyword triggers.
	fields := strings.Fields(lower)
	kept := make([]string, 0, len(fields))
	for _, f := range fields {
		switch {
		case movieKeywords[f]:
			triggered = true
			if mediaType == "" {
				mediaType = "movie"
			}
		case tvKeywords[f]:
			triggered = true
			if mediaType == "" {
				mediaType = "tv"
			}
		case genericMediaKeywords[f]:
			triggered = true
		default:
			kept = append(kept, f)
		}
	}

	if !triggered {
		return Intent{}, false
	}

	// Rebuild the title from the ORIGINAL-cased query, dropping the same tokens
	// we stripped (so "The Dune movie" keeps "The Dune", not "the dune").
	title := rebuildTitle(q, kept)
	if strings.TrimSpace(title) == "" {
		return Intent{}, false
	}
	return Intent{Title: strings.TrimSpace(title), MediaType: mediaType}, true
}

// rebuildTitle reconstructs the title using original casing by keeping only the
// query tokens whose lowercase form survived keyword stripping.
func rebuildTitle(original string, keptLower []string) string {
	want := map[string]int{}
	for _, k := range keptLower {
		want[k]++
	}
	var out []string
	for _, tok := range strings.Fields(original) {
		lt := strings.ToLower(tok)
		if want[lt] > 0 {
			out = append(out, tok)
			want[lt]--
		}
	}
	return strings.Join(out, " ")
}
