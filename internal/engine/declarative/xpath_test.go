package declarative

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/result"
)

func mojeekEngine(t *testing.T) *XPathEngine {
	t.Helper()
	e, err := NewXPath(XPathConfig{
		Name:            "mojeek-declarative",
		SearchURL:       "https://www.mojeek.com/search?q={query}",
		Paging:          true,
		PageSize:        10,
		ResultsXPath:    `//ul[@class="results-standard"]/li/a[@class="ob"]`,
		URLXPath:        "./@href",
		TitleXPath:      "../h2/a",
		ContentXPath:    `..//p[@class="s"]`,
		SuggestionXPath: `//div[@class="top-info"]/p[@class="top-info spell"]/em/a`,
	})
	if err != nil {
		t.Fatalf("new xpath: %v", err)
	}
	return e
}

func TestXPathRequest(t *testing.T) {
	e := mojeekEngine(t)
	req, err := e.Request(context.Background(), engine.Query{Query: "go programming", PageNo: 1, Locale: "all"})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if !strings.Contains(req.URL, "q=go+programming") {
		t.Errorf("query not encoded into url: %s", req.URL)
	}
}

func TestXPathResponseParity(t *testing.T) {
	e := mojeekEngine(t)
	// Reuse the script tier's fixture (same Mojeek HTML).
	body, err := os.ReadFile(filepath.Join("..", "script", "testdata", "mojeek.html"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	res, err := e.Response(context.Background(), &engine.HTTPResponse{StatusCode: 200, Body: body})
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
		t.Fatalf("want 3 results, got %d", len(mains))
	}
	if mains[0].URL != "https://go.dev/" || mains[0].Title != "The Go Programming Language" {
		t.Errorf("first result wrong: %+v", mains[0])
	}
	if mains[0].Content != "Go is an open source programming language supported by Google." {
		t.Errorf("content normalization wrong: %q", mains[0].Content)
	}
	if len(suggs) != 1 || suggs[0].Value != "golang" {
		t.Errorf("suggestion wrong: %+v", suggs)
	}
}
