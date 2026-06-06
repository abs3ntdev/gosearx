// Package media implements the movie/TV knowledge-panel feature: it detects a
// film/TV lookup in a query and assembles a rich card (poster, trailer,
// where-to-watch, ratings, cast) from TMDB (primary) and OMDb (IMDb/Rotten
// Tomatoes/Metacritic ratings). Mirrors the pluggable finance feature.
package media

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/result"
)

// Config configures the media service.
type Config struct {
	TMDBKey string // required; from https://www.themoviedb.org/settings/api
	OMDbKey string // optional; enables IMDb/RT/Metacritic ratings
	Region  string // ISO-3166-1 country for where-to-watch (default "US")
	Timeout time.Duration
}

// Service resolves a query into a Movie knowledge panel.
type Service struct {
	tmdb   *tmdbClient
	omdb   *omdbClient
	region string
	hc     *http.Client
}

// New builds a media Service. Returns an error if no TMDB key is configured
// (TMDB is required; OMDb is optional).
func New(c Config) (*Service, error) {
	if strings.TrimSpace(c.TMDBKey) == "" {
		return nil, fmt.Errorf("media: tmdb_key is required")
	}
	if c.Region == "" {
		c.Region = "US"
	}
	if c.Timeout <= 0 {
		c.Timeout = 6 * time.Second
	}
	hc := &http.Client{Timeout: c.Timeout}
	return &Service{
		tmdb:   newTMDB(c.TMDBKey, hc),
		omdb:   newOMDb(c.OMDbKey, hc),
		region: strings.ToUpper(c.Region),
		hc:     hc,
	}, nil
}

// Lookup detects movie/TV intent and returns a populated Movie, or nil if the
// query isn't a media lookup or nothing matched.
func (s *Service) Lookup(ctx context.Context, query string) (*result.Movie, error) {
	intent, ok := Detect(query)
	if !ok {
		return nil, nil
	}
	id, mediaType, found, err := s.tmdb.searchBest(ctx, intent.Title, intent.MediaType)
	if err != nil || !found {
		return nil, err
	}
	d, err := s.tmdb.detail(ctx, id, mediaType)
	if err != nil {
		return nil, err
	}
	return s.assemble(ctx, d, mediaType), nil
}

func (s *Service) assemble(ctx context.Context, d *tmdbDetail, mediaType string) *result.Movie {
	title := d.Title
	date := d.ReleaseDate
	if mediaType == "tv" {
		title = d.Name
		date = d.FirstAirDate
	}

	m := &result.Movie{
		Type:       result.TypeMovie,
		EngineName: "media",
		MediaType:  mediaType,
		Title:      title,
		Year:       yearOf(date),
		Tagline:    d.Tagline,
		Overview:   d.Overview,
		Poster:     tmdbImageURL(d.PosterPath, "w342"),
		Backdrop:   tmdbImageURL(d.BackdropPath, "w780"),
		Runtime:    runtimeStr(d),
		Status:     d.Status,
		Seasons:    d.NumberOfSeasons,
		Episodes:   d.NumberOfEpisodes,
		TrailerURL: bestTrailer(d),
		URL:        fmt.Sprintf("https://www.themoviedb.org/%s/%d", mediaType, d.ID),
	}
	for _, g := range d.Genres {
		m.Genres = append(m.Genres, g.Name)
	}

	// Directors (movies) / creators (tv).
	if mediaType == "tv" {
		for _, cb := range d.CreatedBy {
			m.Directors = append(m.Directors, cb.Name)
		}
	} else {
		for _, c := range d.Credits.Crew {
			if c.Job == "Director" {
				m.Directors = append(m.Directors, c.Name)
			}
		}
	}

	// Top billed cast (cap at 10).
	for i, c := range d.Credits.Cast {
		if i >= 10 {
			break
		}
		m.Cast = append(m.Cast, result.CastMember{
			Name:      c.Name,
			Character: c.Character,
			Photo:     tmdbImageURL(c.ProfilePath, "w185"),
		})
	}

	// Where to watch for the configured region.
	if pr, ok := d.WatchProviders.Results[s.region]; ok {
		m.ProviderRegion = s.region
		m.JustWatchURL = pr.Link
		add := func(list []tmdbProvider, kind string) {
			for _, p := range list {
				m.Providers = append(m.Providers, result.WatchProvider{
					Name: p.ProviderName,
					Logo: tmdbImageURL(p.LogoPath, "w92"),
					Type: kind,
				})
			}
		}
		add(pr.Flatrate, "stream")
		add(pr.Free, "free")
		add(pr.Ads, "ads")
		add(pr.Rent, "rent")
		add(pr.Buy, "buy")
	}

	// Ratings: TMDB always; IMDb id resolves OMDb for IMDb/RT/Metacritic.
	imdbID := d.IMDbID
	if imdbID == "" {
		imdbID = d.ExternalIDs.IMDbID
	}
	if imdbID != "" {
		m.IMDbURL = "https://www.imdb.com/title/" + imdbID
	}
	if omdb := s.omdb.ratingsByIMDb(ctx, imdbID); omdb != nil {
		for _, src := range []string{"IMDb", "Rotten Tomatoes", "Metacritic"} {
			if v, ok := omdb[src]; ok {
				m.Ratings = append(m.Ratings, result.Rating{Source: src, Value: v})
			}
		}
	}
	if d.VoteAverage > 0 {
		m.Ratings = append(m.Ratings, result.Rating{
			Source: "TMDB",
			Value:  strconv.FormatFloat(d.VoteAverage, 'f', 1, 64) + "/10",
		})
	}

	return m
}
