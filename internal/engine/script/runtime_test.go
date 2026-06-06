package script

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/result"
)

func loadMojeek(t *testing.T) *LuaEngine {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "engines", "mojeek.lua"))
	if err != nil {
		t.Fatalf("read engine: %v", err)
	}
	e, err := Compile("mojeek", string(src))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return e
}

func TestMojeekRequest(t *testing.T) {
	e := loadMojeek(t)
	defer e.Close()

	req, err := e.Request(context.Background(), engine.Query{
		Query: "go programming", PageNo: 2, SafeSearch: 2,
	})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	// safesearch clamped to 1, page 2 -> s=10, query percent-encoded.
	if !strings.Contains(req.URL, "mojeek.com/search?") {
		t.Fatalf("unexpected url: %s", req.URL)
	}
	if !strings.Contains(req.URL, "safe=1") {
		t.Errorf("safesearch not clamped to 1: %s", req.URL)
	}
	if !strings.Contains(req.URL, "s=10") {
		t.Errorf("page offset missing: %s", req.URL)
	}
	if !strings.Contains(req.URL, "q=go+programming") {
		t.Errorf("query missing/misencoded: %s", req.URL)
	}
}

func TestMojeekResponseParity(t *testing.T) {
	e := loadMojeek(t)
	defer e.Close()

	body, err := os.ReadFile(filepath.Join("testdata", "mojeek.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	res, err := e.Response(context.Background(), &engine.HTTPResponse{
		StatusCode: 200, URL: "https://www.mojeek.com/search?q=go", Body: body,
	})
	if err != nil {
		t.Fatalf("response: %v", err)
	}

	var mains []*result.MainResult
	var suggs []*result.Suggestion
	for _, r := range res {
		switch v := r.(type) {
		case *result.MainResult:
			mains = append(mains, v)
		case *result.Suggestion:
			suggs = append(suggs, v)
		}
	}

	if len(mains) != 3 {
		t.Fatalf("want 3 main results, got %d: %+v", len(mains), mains)
	}

	// Validate the nested/relative XPath worked: title from ../h2/a,
	// content from ..//p[@class="s"], url from ./@href.
	first := mains[0]
	if first.URL != "https://go.dev/" {
		t.Errorf("url xpath (./@href) wrong: %q", first.URL)
	}
	if first.Title != "The Go Programming Language" {
		t.Errorf("title xpath (../h2/a) wrong: %q", first.Title)
	}
	// normalizeSpace must collapse the double space in the fixture.
	if first.Content != "Go is an open source programming language supported by Google." {
		t.Errorf("content xpath/normalize wrong: %q", first.Content)
	}
	if first.Engine() != "mojeek" {
		t.Errorf("engine name not propagated: %q", first.Engine())
	}

	// Suggestion extracted via //div[@class='top-info']...//em/a
	if len(suggs) != 1 || suggs[0].Value != "golang" {
		t.Errorf("suggestion parsing wrong: %+v", suggs)
	}
}

// TestConcurrentUse stresses the LState pool: many goroutines hitting the same
// engine concurrently must not race or corrupt state. Run with -race.
func TestConcurrentUse(t *testing.T) {
	e := loadMojeek(t)
	defer e.Close()
	body, _ := os.ReadFile(filepath.Join("testdata", "mojeek.html"))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := e.Response(context.Background(), &engine.HTTPResponse{
				StatusCode: 200, Body: body,
			})
			if err != nil {
				t.Errorf("concurrent response: %v", err)
				return
			}
			if len(res) != 4 { // 3 main + 1 suggestion
				t.Errorf("concurrent: want 4 results, got %d", len(res))
			}
		}()
	}
	wg.Wait()
}

// TestContextDeadline verifies the context deadline is wired into the VM.
func TestContextDeadline(t *testing.T) {
	// An engine with an infinite loop should be killed by the context.
	src := `
function request(query, params) params.url = "x"; return params end
function response(resp)
  local results = {}
  while true do results[#results+1] = 1 end
  return results
end`
	e, err := Compile("loop", src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = e.Response(ctx, &engine.HTTPResponse{StatusCode: 200, Body: []byte("")})
	if err == nil {
		t.Fatal("expected context deadline error, got nil")
	}
}
