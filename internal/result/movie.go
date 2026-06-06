package result

// movie.go defines the Movie knowledge-panel result type: a rich card for a film
// or TV show (poster, trailer, where-to-watch, ratings, cast) — the gosearx
// analog of Google's media knowledge panel. Populated by internal/media.

// Movie is a rich film/TV result.
type Movie struct {
	Type       Type   `json:"type"`
	EngineName string `json:"engine"`

	MediaType string   `json:"mediaType"` // "movie" | "tv"
	Title     string   `json:"title"`
	Year      string   `json:"year,omitempty"`
	Tagline   string   `json:"tagline,omitempty"`
	Overview  string   `json:"overview,omitempty"`
	Poster    string   `json:"poster,omitempty"`   // absolute image URL
	Backdrop  string   `json:"backdrop,omitempty"` // absolute image URL
	Runtime   string   `json:"runtime,omitempty"`  // e.g. "2h 16m" or "45m/ep"
	Genres    []string `json:"genres,omitempty"`
	Status    string   `json:"status,omitempty"` // e.g. "Released", "Returning Series"
	Seasons   int      `json:"seasons,omitempty"`
	Episodes  int      `json:"episodes,omitempty"`

	// TrailerURL is a watchable YouTube trailer link, if available.
	TrailerURL string `json:"trailerUrl,omitempty"`

	// Ratings from various sources (IMDb, Rotten Tomatoes, Metacritic, TMDB).
	Ratings []Rating `json:"ratings,omitempty"`

	// Cast is the top billed cast.
	Cast []CastMember `json:"cast,omitempty"`

	// Directors/Creators for the credit line.
	Directors []string `json:"directors,omitempty"`

	// Providers are where-to-watch options for the configured region.
	Providers []WatchProvider `json:"providers,omitempty"`
	// ProviderRegion is the ISO country the providers apply to (e.g. "US").
	ProviderRegion string `json:"providerRegion,omitempty"`
	// JustWatchURL deep-links to the full where-to-watch listing.
	JustWatchURL string `json:"justWatchUrl,omitempty"`

	// URL is the canonical info page (TMDB or IMDb).
	URL     string `json:"url,omitempty"`
	IMDbURL string `json:"imdbUrl,omitempty"`
}

// Rating is a score from one source.
type Rating struct {
	Source string `json:"source"` // "IMDb" | "Rotten Tomatoes" | "Metacritic" | "TMDB"
	Value  string `json:"value"`  // e.g. "8.6/10", "91%", "78"
}

// CastMember is one actor + their role.
type CastMember struct {
	Name      string `json:"name"`
	Character string `json:"character,omitempty"`
	Photo     string `json:"photo,omitempty"` // absolute image URL
}

// WatchProvider is one streaming/rental/purchase option.
type WatchProvider struct {
	Name string `json:"name"`
	Logo string `json:"logo,omitempty"` // absolute image URL
	Type string `json:"type"`           // "stream" | "rent" | "buy" | "free" | "ads"
}

func (m *Movie) Kind() Type     { return TypeMovie }
func (m *Movie) Engine() string { return m.EngineName }
