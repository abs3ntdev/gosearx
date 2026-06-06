package media

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// omdbClient fetches IMDb / Rotten Tomatoes / Metacritic ratings by IMDb ID.
// OMDb is the practical way to get Rotten Tomatoes (which has no free API).
type omdbClient struct {
	key string
	hc  *http.Client
}

func newOMDb(key string, hc *http.Client) *omdbClient {
	return &omdbClient{key: key, hc: hc}
}

type omdbResponse struct {
	Response   string `json:"Response"`
	IMDbRating string `json:"imdbRating"`
	IMDbVotes  string `json:"imdbVotes"`
	Ratings    []struct {
		Source string `json:"Source"`
		Value  string `json:"Value"`
	} `json:"Ratings"`
}

// ratingsByIMDb returns (source -> value) ratings for an IMDb id, or nil.
func (c *omdbClient) ratingsByIMDb(ctx context.Context, imdbID string) map[string]string {
	if c == nil || c.key == "" || imdbID == "" {
		return nil
	}
	u := "https://www.omdbapi.com/?" + url.Values{
		"i":      {imdbID},
		"apikey": {c.key},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, httpLimit))
	if err != nil {
		return nil
	}
	var o omdbResponse
	if json.Unmarshal(body, &o) != nil || o.Response == "False" {
		return nil
	}
	out := map[string]string{}
	if o.IMDbRating != "" && o.IMDbRating != "N/A" {
		out["IMDb"] = o.IMDbRating + "/10"
	}
	for _, r := range o.Ratings {
		switch r.Source {
		case "Rotten Tomatoes":
			out["Rotten Tomatoes"] = r.Value
		case "Metacritic":
			out["Metacritic"] = r.Value
		}
	}
	return out
}
