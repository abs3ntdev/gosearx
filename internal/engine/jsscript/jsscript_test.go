package jsscript

import (
	"context"
	"testing"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/result"
)

func TestJSEngine_RequestResponse(t *testing.T) {
	src := `
		function request(query, params) {
			params.url = "https://api.example.com/search?" + url.encode({ q: query });
			params.headers["X-Test"] = "1";
			return params;
		}
		function response(resp) {
			var data = JSON.parse(resp.text);
			return data.items.map(function(it) {
				return { title: it.name, url: it.link, content: it.desc };
			});
		}`
	e, err := Compile("ex", src)
	if err != nil {
		t.Fatal(err)
	}
	req, err := e.Request(context.Background(), engine.Query{Query: "hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.example.com/search?q=hello+world" {
		t.Fatalf("url = %q", req.URL)
	}
	if req.Headers["X-Test"] != "1" {
		t.Fatalf("headers = %v", req.Headers)
	}
	body := `{"items":[{"name":"A","link":"https://a.com","desc":"x"},{"name":"B","link":"https://b.com","desc":"y"}]}`
	res, err := e.Response(context.Background(), &engine.HTTPResponse{Body: []byte(body), StatusCode: 200})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("want 2, got %d", len(res))
	}
	mr, ok := res[0].(*result.MainResult)
	if !ok || mr.Title != "A" || mr.URL != "https://a.com" {
		t.Fatalf("got %#v", res[0])
	}
}
