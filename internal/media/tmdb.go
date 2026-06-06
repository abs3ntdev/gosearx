package media

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	tmdbBase  = "https://api.themoviedb.org/3"
	tmdbImg   = "https://image.tmdb.org/t/p"
	httpLimit = 1 << 20
)

// tmdbClient talks to The Movie Database API.
type tmdbClient struct {
	key string
	hc  *http.Client
}

func newTMDB(key string, hc *http.Client) *tmdbClient {
	return &tmdbClient{key: key, hc: hc}
}

func (c *tmdbClient) get(ctx context.Context, path string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("api_key", c.key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tmdbBase+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tmdb %s: status %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, httpLimit))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// --- search/multi to find the best title match ---

type tmdbSearchResult struct {
	Results []struct {
		ID           int     `json:"id"`
		MediaType    string  `json:"media_type"`
		Title        string  `json:"title"`          // movies
		Name         string  `json:"name"`           // tv
		ReleaseDate  string  `json:"release_date"`   // movies
		FirstAirDate string  `json:"first_air_date"` // tv
		Popularity   float64 `json:"popularity"`
	} `json:"results"`
}

// searchBest returns the (id, mediaType) of the best match, biasing toward
// typeHint ("movie"/"tv"/"") and popularity. Returns ok=false if nothing found.
func (c *tmdbClient) searchBest(ctx context.Context, title, typeHint string) (int, string, bool, error) {
	var sr tmdbSearchResult
	if err := c.get(ctx, "/search/multi", url.Values{
		"query":         {title},
		"include_adult": {"false"},
	}, &sr); err != nil {
		return 0, "", false, err
	}
	bestID, bestType := 0, ""
	bestScore := -1.0
	for _, r := range sr.Results {
		if r.MediaType != "movie" && r.MediaType != "tv" {
			continue // skip persons, etc.
		}
		score := r.Popularity
		if typeHint != "" && r.MediaType == typeHint {
			score += 1000 // strongly prefer the hinted type
		}
		// Exact (case-insensitive) title match gets a big boost.
		name := r.Title
		if name == "" {
			name = r.Name
		}
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(title)) {
			score += 5000
		}
		if score > bestScore {
			bestScore = score
			bestID = r.ID
			bestType = r.MediaType
		}
	}
	if bestID == 0 {
		return 0, "", false, nil
	}
	return bestID, bestType, true, nil
}

// --- full detail (append videos, credits, watch providers, external ids) ---

type tmdbDetail struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	Name             string  `json:"name"`
	Tagline          string  `json:"tagline"`
	Overview         string  `json:"overview"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	ReleaseDate      string  `json:"release_date"`
	FirstAirDate     string  `json:"first_air_date"`
	Runtime          int     `json:"runtime"`          // movies (minutes)
	EpisodeRunTime   []int   `json:"episode_run_time"` // tv
	Status           string  `json:"status"`
	VoteAverage      float64 `json:"vote_average"`
	NumberOfSeasons  int     `json:"number_of_seasons"`
	NumberOfEpisodes int     `json:"number_of_episodes"`
	Genres           []struct {
		Name string `json:"name"`
	} `json:"genres"`
	Videos struct {
		Results []struct {
			Site     string `json:"site"`
			Type     string `json:"type"`
			Key      string `json:"key"`
			Official bool   `json:"official"`
		} `json:"results"`
	} `json:"videos"`
	Credits struct {
		Cast []struct {
			Name        string `json:"name"`
			Character   string `json:"character"`
			ProfilePath string `json:"profile_path"`
		} `json:"cast"`
		Crew []struct {
			Name string `json:"name"`
			Job  string `json:"job"`
		} `json:"crew"`
	} `json:"credits"`
	CreatedBy []struct {
		Name string `json:"name"`
	} `json:"created_by"`
	ExternalIDs struct {
		IMDbID string `json:"imdb_id"`
	} `json:"external_ids"`
	IMDbID         string `json:"imdb_id"` // present on movie detail directly
	WatchProviders struct {
		Results map[string]tmdbProviderRegion `json:"results"`
	} `json:"watch/providers"`
}

type tmdbProviderRegion struct {
	Link     string         `json:"link"`
	Flatrate []tmdbProvider `json:"flatrate"`
	Rent     []tmdbProvider `json:"rent"`
	Buy      []tmdbProvider `json:"buy"`
	Free     []tmdbProvider `json:"free"`
	Ads      []tmdbProvider `json:"ads"`
}

type tmdbProvider struct {
	ProviderName string `json:"provider_name"`
	LogoPath     string `json:"logo_path"`
}

func (c *tmdbClient) detail(ctx context.Context, id int, mediaType string) (*tmdbDetail, error) {
	var d tmdbDetail
	if err := c.get(ctx, fmt.Sprintf("/%s/%d", mediaType, id), url.Values{
		"append_to_response": {"videos,credits,watch/providers,external_ids"},
	}, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func tmdbImageURL(path, size string) string {
	if path == "" {
		return ""
	}
	return tmdbImg + "/" + size + path
}

func runtimeStr(d *tmdbDetail) string {
	if d.Runtime > 0 {
		h, m := d.Runtime/60, d.Runtime%60
		if h > 0 {
			return fmt.Sprintf("%dh %dm", h, m)
		}
		return fmt.Sprintf("%dm", m)
	}
	if len(d.EpisodeRunTime) > 0 && d.EpisodeRunTime[0] > 0 {
		return strconv.Itoa(d.EpisodeRunTime[0]) + "m/ep"
	}
	return ""
}

func yearOf(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}

// bestTrailer picks an official YouTube trailer (falling back to any YouTube
// trailer/teaser).
func bestTrailer(d *tmdbDetail) string {
	var fallback string
	for _, v := range d.Videos.Results {
		if v.Site != "YouTube" {
			continue
		}
		url := "https://www.youtube.com/watch?v=" + v.Key
		if v.Type == "Trailer" && v.Official {
			return url
		}
		if fallback == "" && (v.Type == "Trailer" || v.Type == "Teaser") {
			fallback = url
		}
	}
	return fallback
}

var _ = time.Second
