package search

import (
	"context"
	"testing"
	"time"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/result"
)

// fakeEngine returns predefined results for the merge/score tests.
type fakeEngine struct {
	name    string
	results result.EngineResults
	delay   time.Duration
	reqErr  error
}

func (e *fakeEngine) Name() string { return e.name }
func (e *fakeEngine) Request(_ context.Context, _ engine.Query) (*engine.HTTPRequest, error) {
	if e.reqErr != nil {
		return nil, e.reqErr
	}
	return &engine.HTTPRequest{URL: "https://example.test/"}, nil
}
func (e *fakeEngine) Response(ctx context.Context, _ *engine.HTTPResponse) (result.EngineResults, error) {
	if e.delay > 0 {
		select {
		case <-time.After(e.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return e.results, nil
}

// fakeFetcher returns an empty response (engines here ignore the body).
type fakeFetcher struct{}

func (fakeFetcher) Fetch(_ context.Context, _ *engine.HTTPRequest) (*engine.HTTPResponse, error) {
	return &engine.HTTPResponse{StatusCode: 200, Body: []byte("")}, nil
}

func mr(eng, title, url string) *result.MainResult {
	return &result.MainResult{Type: result.TypeMain, EngineName: eng, Title: title, URL: url}
}

func reg(e engine.Engine, weight float64) *engine.Registered {
	return &engine.Registered{
		Engine: e,
		Meta:   engine.Meta{Name: e.Name(), Timeout: 2 * time.Second, Weight: weight},
	}
}

func TestMergeDedupAndScore(t *testing.T) {
	// Two engines, overlapping URL "go.dev" (should merge, gain 2 positions).
	e1 := &fakeEngine{name: "alpha", results: result.EngineResults{
		mr("alpha", "Go", "https://go.dev/"),          // pos 1
		mr("alpha", "Rust", "https://rust-lang.org/"), // pos 2
	}}
	e2 := &fakeEngine{name: "beta", results: result.EngineResults{
		mr("beta", "The Go Programming Language", "https://go.dev/"), // pos 1, dup of go.dev
		mr("beta", "Zig", "https://ziglang.org/"),                    // pos 2
	}}

	s := New(fakeFetcher{})
	resp, err := s.Search(context.Background(), Query{
		Text:    "langs",
		Engines: []*engine.Registered{reg(e1, 1), reg(e2, 1)},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	// 3 unique results (go.dev merged).
	if len(resp.Results) != 3 {
		t.Fatalf("want 3 merged results, got %d", len(resp.Results))
	}

	// Top result must be go.dev: it has 2 positions from 2 engines.
	top := resp.Results[0]
	if top.URL != "https://go.dev/" {
		t.Errorf("expected go.dev on top, got %s (score %.3f)", top.URL, top.Score)
	}
	if len(top.Engines) != 2 {
		t.Errorf("merged result should list 2 engines, got %v", top.Engines)
	}
	// Longer title should win the merge.
	if top.Title != "The Go Programming Language" {
		t.Errorf("merge should keep longer title, got %q", top.Title)
	}
	// score = weight(=1*2 positions) * (1/1 + 1/1) = 2 * 2 = 4.
	if top.Score != 4 {
		t.Errorf("expected merged score 4, got %.3f", top.Score)
	}
}

func TestEngineTimeoutIsolated(t *testing.T) {
	fast := &fakeEngine{name: "fast", results: result.EngineResults{mr("fast", "ok", "https://ok.test/")}}
	slow := &fakeEngine{name: "slow", delay: time.Second, results: result.EngineResults{mr("slow", "late", "https://late.test/")}}

	slowReg := reg(slow, 1)
	slowReg.Meta.Timeout = 50 * time.Millisecond // force timeout

	s := New(fakeFetcher{})
	resp, err := s.Search(context.Background(), Query{
		Text:    "x",
		Engines: []*engine.Registered{reg(fast, 1), slowReg},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].URL != "https://ok.test/" {
		t.Errorf("fast engine result missing: %+v", resp.Results)
	}
	if len(resp.Unresponsive) != 1 || resp.Unresponsive[0].Engine != "slow" {
		t.Errorf("slow engine should be marked unresponsive: %+v", resp.Unresponsive)
	}
}

func TestEngineErrorIsolated(t *testing.T) {
	good := &fakeEngine{name: "good", results: result.EngineResults{mr("good", "ok", "https://ok.test/")}}
	bad := &fakeEngine{name: "bad", reqErr: context.Canceled}

	s := New(fakeFetcher{})
	resp, err := s.Search(context.Background(), Query{
		Text:    "x",
		Engines: []*engine.Registered{reg(good, 1), reg(bad, 1)},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Errorf("good engine should still return: %+v", resp.Results)
	}
	if len(resp.Unresponsive) != 1 {
		t.Errorf("bad engine should be unresponsive: %+v", resp.Unresponsive)
	}
}
