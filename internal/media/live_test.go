package media

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLive exercises the real TMDB (+ optional OMDb) APIs end-to-end. It is
// skipped unless TMDB_API_KEY is set, so it never runs in CI and never needs
// secrets committed. Run locally:
//
//	TMDB_API_KEY=... OMDB_API_KEY=... go test ./internal/media/ -run TestLive -v -count=1
func TestLive(t *testing.T) {
	tmdb := os.Getenv("TMDB_API_KEY")
	if tmdb == "" {
		t.Skip("set TMDB_API_KEY to run the live media test")
	}
	svc, err := New(Config{
		TMDBKey: tmdb,
		OMDbKey: os.Getenv("OMDB_API_KEY"),
		Region:  "US",
		Timeout: 8 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	for _, q := range []string{"dune movie", "the office tv show", "oppenheimer trailer"} {
		m, err := svc.Lookup(ctx, q)
		if err != nil {
			t.Errorf("%q: error: %v", q, err)
			continue
		}
		if m == nil {
			t.Errorf("%q: no movie returned", q)
			continue
		}
		t.Logf("%q -> %q (%s) type=%s", q, m.Title, m.Year, m.MediaType)
		t.Logf("    poster=%v trailer=%v providers=%d cast=%d ratings=%d",
			m.Poster != "", m.TrailerURL != "", len(m.Providers), len(m.Cast), len(m.Ratings))
		for _, r := range m.Ratings {
			t.Logf("    rating %s = %s", r.Source, r.Value)
		}
		if m.Title == "" || m.Poster == "" {
			t.Errorf("%q: missing core fields (title/poster)", q)
		}
	}
}
