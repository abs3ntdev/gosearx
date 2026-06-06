package declarative

import (
	"context"
	"testing"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/result"
)

const mdnFixture = `{
  "documents": [
    {"mdn_url": "/en-US/docs/Web/JavaScript", "title": "JavaScript", "summary": "JS docs"},
    {"mdn_url": "/en-US/docs/Web/CSS", "title": "CSS", "summary": "CSS docs"}
  ],
  "suggestions": ["javascript guide"]
}`

func TestJSONEngineParse(t *testing.T) {
	e, err := NewJSON(JSONConfig{
		Name:            "mdn",
		SearchURL:       "https://developer.mozilla.org/api/v1/search?q={query}&page={pageno}",
		Paging:          true,
		ResultsQuery:    "documents",
		URLQuery:        "mdn_url",
		URLPrefix:       "https://developer.mozilla.org",
		TitleQuery:      "title",
		ContentQuery:    "summary",
		SuggestionQuery: "suggestions",
	})
	if err != nil {
		t.Fatalf("new json: %v", err)
	}

	res, err := e.Response(context.Background(), &engine.HTTPResponse{
		StatusCode: 200, Body: []byte(mdnFixture),
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
	if len(mains) != 2 {
		t.Fatalf("want 2 results, got %d", len(mains))
	}
	if mains[0].URL != "https://developer.mozilla.org/en-US/docs/Web/JavaScript" {
		t.Errorf("url_prefix not applied: %q", mains[0].URL)
	}
	if mains[0].Title != "JavaScript" || mains[0].Content != "JS docs" {
		t.Errorf("title/content wrong: %+v", mains[0])
	}
	if len(suggs) != 1 || suggs[0].Value != "javascript guide" {
		t.Errorf("suggestion wrong: %+v", suggs)
	}
}

func TestJSONRequestURL(t *testing.T) {
	e, _ := NewJSON(JSONConfig{
		Name: "mdn", SearchURL: "https://x/api?q={query}&page={pageno}",
		URLQuery: "u", TitleQuery: "t", Paging: true,
	})
	req, err := e.Request(context.Background(), engine.Query{Query: "a b", PageNo: 2, Locale: "all"})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	want := "https://x/api?q=a+b&page=2"
	if req.URL != want {
		t.Errorf("url = %q, want %q", req.URL, want)
	}
}
