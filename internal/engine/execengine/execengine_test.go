package execengine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/result"
)

func TestExecEngine_RequestResponse(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no shell")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "e.sh")
	// A trivial two-phase engine: request -> url; response -> one result.
	body := `#!/bin/sh
read payload
case "$payload" in
  *'"phase":"request"'*)
    printf '{"url":"https://example.com/q","headers":{"A":"b"}}' ;;
  *'"phase":"response"'*)
    printf '{"results":[{"title":"hi","url":"https://x.com","content":"c"}]}' ;;
esac
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	e, err := Compile("e", script)
	if err != nil {
		t.Fatal(err)
	}
	req, err := e.Request(context.Background(), engine.Query{Query: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if req == nil || req.URL != "https://example.com/q" || req.Headers["A"] != "b" {
		t.Fatalf("req = %#v", req)
	}
	res, err := e.Response(context.Background(), &engine.HTTPResponse{StatusCode: 200, Body: []byte("ignored")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("want 1, got %d", len(res))
	}
	if mr, ok := res[0].(*result.MainResult); !ok || mr.Title != "hi" {
		t.Fatalf("got %#v", res[0])
	}
}
