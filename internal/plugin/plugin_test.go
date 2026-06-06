package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/searxng/gosearx/internal/result"
)

func pluginsDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "plugins")
}

func TestLoadBuiltinPlugins(t *testing.T) {
	store, errs := LoadDir(pluginsDir(t))
	for _, e := range errs {
		t.Errorf("load error: %v", e)
	}
	names := store.Names()
	if len(names) == 0 {
		t.Fatal("no plugins loaded")
	}
	t.Logf("loaded plugins: %v", names)
}

func TestCalculatorPlugin(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(pluginsDir(t), "calculator.lua"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := CompileLua("calculator", string(src))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	cases := map[string]string{
		"2 + 3 * 4":   "2 + 3 * 4 = 14",
		"(2 + 3) * 4": "(2 + 3) * 4 = 20",
		"10 / 4":      "10 / 4 = 2.5",
		"2 ^ 10":      "2 ^ 10 = 1024",
		"-5 + 2":      "-5 + 2 = -3",
	}
	for expr, want := range cases {
		res, err := p.PostSearch(&SearchContext{Query: expr})
		if err != nil {
			t.Errorf("%q: %v", expr, err)
			continue
		}
		if len(res) != 1 {
			t.Errorf("%q: want 1 answer, got %d", expr, len(res))
			continue
		}
		ans, ok := res[0].(*result.Answer)
		if !ok || ans.Answer != want {
			t.Errorf("%q: got %v, want %q", expr, res[0], want)
		}
	}

	// Non-math queries produce nothing.
	res, _ := p.PostSearch(&SearchContext{Query: "hello world"})
	if len(res) != 0 {
		t.Errorf("non-math query produced results: %v", res)
	}
}

func TestTrackerRemoverPlugin(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(pluginsDir(t), "tracker_remover.lua"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := CompileLua("tracker_remover", string(src))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	r := &result.MainResult{URL: "https://example.com/page?id=5&utm_source=x&utm_medium=y"}
	keep, err := p.OnResult(r)
	if err != nil {
		t.Fatal(err)
	}
	if !keep {
		t.Fatal("result should be kept")
	}
	if r.URL != "https://example.com/page?id=5" {
		t.Errorf("trackers not stripped: %q", r.URL)
	}
}

func TestKeywordGating(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(pluginsDir(t), "hash.lua"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := CompileLua("hash", string(src))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(p.Keywords()) == 0 {
		t.Fatal("expected keywords")
	}
	// Matches keyword.
	if !MatchKeyword(p.Keywords(), "base64 hello") {
		t.Error("should match base64 keyword")
	}
	// Doesn't match.
	if MatchKeyword(p.Keywords(), "regular query") {
		t.Error("should not match")
	}
	res, err := p.PostSearch(&SearchContext{Query: "base64 hello"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("want 1 answer, got %d", len(res))
	}
	ans := res[0].(*result.Answer)
	if ans.Answer != "aGVsbG8=" {
		t.Errorf("base64 wrong: %q", ans.Answer)
	}
}

func TestStorageHooks(t *testing.T) {
	store, _ := LoadDir(pluginsDir(t))

	// pre_search defaults to allow.
	if !store.PreSearch(&SearchContext{Query: "test"}) {
		t.Error("pre_search should allow")
	}

	// on_result: tracker remover strips, keeps result.
	r := &result.MainResult{URL: "https://e.com/?utm_source=x"}
	if !store.OnResult(r) {
		t.Error("result should be kept")
	}
	if r.URL != "https://e.com/" {
		t.Errorf("tracker not stripped via storage: %q", r.URL)
	}

	// post_search: calculator answers a math query.
	added := store.PostSearch(&SearchContext{Query: "7 * 6"})
	found := false
	for _, a := range added {
		if ans, ok := a.(*result.Answer); ok && ans.Answer == "7 * 6 = 42" {
			found = true
		}
	}
	if !found {
		t.Errorf("calculator answer not found in: %v", added)
	}
}
