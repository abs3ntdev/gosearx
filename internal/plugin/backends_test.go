package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/searxng/gosearx/internal/result"
)

func TestJSPlugin_PostSearch(t *testing.T) {
	src := `
		var keywords = ["greet"];
		function postSearch(ctx) {
			return [{ type: "answer", answer: "hello " + ctx.query }];
		}`
	p, err := CompileJS("greet", src)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Keywords(); len(got) != 1 || got[0] != "greet" {
		t.Fatalf("keywords = %v", got)
	}
	res, err := p.PostSearch(&SearchContext{Query: "greet world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("want 1 result, got %d", len(res))
	}
	a, ok := res[0].(*result.Answer)
	if !ok || a.Answer != "hello greet world" {
		t.Fatalf("got %#v", res[0])
	}
}

func TestJSPlugin_OnResult(t *testing.T) {
	// strip a tracking param by rewriting url
	src := `
		function onResult(r) {
			r.url = r.url.replace(/\?.*$/, "");
			return r;
		}`
	p, err := CompileJS("clean", src)
	if err != nil {
		t.Fatal(err)
	}
	mr := &result.MainResult{Title: "x", URL: "https://e.com/p?utm=1", EngineName: "e"}
	keep, err := p.OnResult(mr)
	if err != nil || !keep {
		t.Fatalf("keep=%v err=%v", keep, err)
	}
	if mr.URL != "https://e.com/p" {
		t.Fatalf("url = %q", mr.URL)
	}
}

func TestJSPlugin_PreSearchAbort(t *testing.T) {
	p, err := CompileJS("blk", `function preSearch(ctx){ return false; }`)
	if err != nil {
		t.Fatal(err)
	}
	allow, err := p.PreSearch(&SearchContext{Query: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if allow {
		t.Fatal("expected abort")
	}
}

func TestExecPlugin_PostSearch(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no /bin/sh")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "echoer.sh")
	body := "#!/bin/sh\n" +
		"# @keywords: echo\n" +
		"read line\n" +
		`printf '{"results":[{"type":"answer","answer":"ok"}]}'` + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := CompileExec(script, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Keywords(); len(got) != 1 || got[0] != "echo" {
		t.Fatalf("keywords = %v", got)
	}
	res, err := p.PostSearch(&SearchContext{Query: "echo hi"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("want 1, got %d", len(res))
	}
	if a, ok := res[0].(*result.Answer); !ok || a.Answer != "ok" {
		t.Fatalf("got %#v", res[0])
	}
}

func TestExecHeaderParse(t *testing.T) {
	p := &ExecPlugin{}
	parseExecHeader("#!/usr/bin/env python3\n# @name: custom\n# @keywords: a, b , c\n# @timeout: 2s\nprint('x')\n", p)
	if p.name != "custom" {
		t.Errorf("name = %q", p.name)
	}
	if len(p.keywords) != 3 || p.keywords[2] != "c" {
		t.Errorf("keywords = %v", p.keywords)
	}
	if p.timeout.Seconds() != 2 {
		t.Errorf("timeout = %v", p.timeout)
	}
}
