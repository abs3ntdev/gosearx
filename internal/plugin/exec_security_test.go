package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/searxng/gosearx/internal/result"
)

func answerText(res result.EngineResults) string {
	if len(res) == 1 {
		if a, ok := res[0].(*result.Answer); ok {
			return a.Answer
		}
	}
	return ""
}

// Proves a malicious search query CANNOT inject commands: the query is delivered
// only as JSON on stdin, never as argv or to `sh -c`. If injection were possible
// the marker file would be created.
func TestExec_NoCommandInjection(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no shell")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "PWNED")
	script := filepath.Join(dir, "echo.sh")
	body := "#!/bin/sh\nread line\nprintf '{\"results\":[{\"type\":\"answer\",\"answer\":\"ok\"}]}'\n"
	os.WriteFile(script, []byte(body), 0o755)
	p, err := CompileExec(script, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Hostile queries that WOULD break a shell if interpolated into argv/`sh -c`.
	for _, q := range []string{
		"; touch " + marker,
		"$(touch " + marker + ")",
		"`touch " + marker + "`",
		"x && touch " + marker,
		"| touch " + marker,
		"\n touch " + marker,
	} {
		if _, err := p.PostSearch(&SearchContext{Query: q}); err != nil {
			t.Fatalf("query %q errored: %v", q, err)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("SECURITY: command injection succeeded — marker file was created")
	}
}

// Proves inherited environment secrets are NOT visible to the plugin.
func TestExec_EnvScrubbed(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no shell")
	}
	os.Setenv("GOSEARX_SECRET_TEST", "topsecret")
	defer os.Unsetenv("GOSEARX_SECRET_TEST")
	dir := t.TempDir()
	script := filepath.Join(dir, "env.sh")
	body := "#!/bin/sh\nread line\nprintf '{\"results\":[{\"type\":\"answer\",\"answer\":\"%s\"}]}' \"${GOSEARX_SECRET_TEST:-NONE}\"\n"
	os.WriteFile(script, []byte(body), 0o755)
	p, _ := CompileExec(script, 0)
	res, err := p.PostSearch(&SearchContext{Query: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got := answerText(res); !strings.Contains(got, "NONE") || strings.Contains(got, "topsecret") {
		t.Fatalf("SECURITY: plugin saw inherited env: %q", got)
	}
}

// Proves a runaway plugin that floods stdout is rejected, not allowed to OOM.
func TestExec_OutputCap(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no shell")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "flood.sh")
	body := "#!/bin/sh\nread line\nyes AAAAAAAAAA | head -c 8000000\n"
	os.WriteFile(script, []byte(body), 0o755)
	p, _ := CompileExec(script, 0)
	_, err := p.PostSearch(&SearchContext{Query: "x"})
	if err == nil || !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("expected output-cap error, got %v", err)
	}
}

// Proves a hung plugin is killed by the timeout instead of blocking forever.
func TestExec_Timeout(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no shell")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "hang.sh")
	os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o755)
	p, _ := CompileExec(script, 500_000_000) // 0.5s
	_, err := p.PostSearch(&SearchContext{Query: "x"})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
