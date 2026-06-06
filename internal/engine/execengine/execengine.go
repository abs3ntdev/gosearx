// Package execengine implements an engine tier backed by an external executable
// (bash, python, node, ruby, a compiled binary…). The gosearx binary stays
// pure-Go/CGO-free; interpreters are the user's and invoked as subprocesses.
//
// The host still performs the HTTP fetch (engines never do I/O themselves), so
// the script is invoked twice per search via a JSON stdin/stdout protocol:
//
//	phase "request":  stdin {"phase":"request","query","pageno","language","safesearch","time_range","config"}
//	                  stdout {"url","method","headers","cookies","data"}
//	phase "response": stdin {"phase":"response","text","url","status_code","query","config"}
//	                  stdout {"results":[ {<result map>}, ... ]}
//
// Result maps use the same schema as every other tier (result.FromMap).
package execengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/result"
)

// maxExecOutput bounds a script's stdout to protect the host from OOM.
const maxExecOutput = 4 << 20

// cappedBuffer stops accumulating after limit bytes and records truncation.
type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.buf.Len() >= c.limit {
		c.truncated = true
		return len(p), nil
	}
	if c.buf.Len()+len(p) > c.limit {
		c.truncated = true
		c.buf.Write(p[:c.limit-c.buf.Len()])
		return len(p), nil
	}
	return c.buf.Write(p)
}

func (c *cappedBuffer) Bytes() []byte  { return c.buf.Bytes() }
func (c *cappedBuffer) String() string { return c.buf.String() }

// ExecEngine is an engine.Engine backed by an external script.
type ExecEngine struct {
	name   string
	path   string
	interp []string
}

var execExtensions = map[string][]string{
	".sh":   {"bash"},
	".bash": {"bash"},
	".py":   {"python3"},
	".rb":   {"ruby"},
	".pl":   {"perl"},
	".js":   {"node"},
}

// IsExecExtension reports whether a file should load as an exec engine.
func IsExecExtension(ext string) bool {
	if ext == ".engine" {
		return true
	}
	_, ok := execExtensions[strings.ToLower(ext)]
	return ok
}

// Compile prepares an exec engine (does not run the script).
func Compile(name, path string) (*ExecEngine, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(abs))
	return &ExecEngine{name: name, path: abs, interp: execExtensions[ext]}, nil
}

func (e *ExecEngine) Name() string { return e.name }

// run invokes the script with the given JSON payload and decodes stdout JSON.
func (e *ExecEngine) run(ctx context.Context, payload any) (map[string]any, error) {
	var cmd *exec.Cmd
	if len(e.interp) > 0 {
		cmd = exec.CommandContext(ctx, e.interp[0], append(e.interp[1:], e.path)...)
	} else {
		cmd = exec.CommandContext(ctx, e.path)
	}
	in, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	cmd.Stdin = bytes.NewReader(in)
	// Hermetic env: no inherited secrets leak to the script.
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "GOSEARX_ENGINE=1"}
	// Own process group + kill the whole group on timeout/cancel.
	cmd.SysProcAttr = detachedSysProcAttr()
	cmd.Cancel = func() error { killProcessGroup(cmd); return nil }
	cmd.WaitDelay = time.Second
	// Bound output so a runaway script can't OOM the host.
	stdout := &cappedBuffer{limit: maxExecOutput}
	stderr := &cappedBuffer{limit: 8 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("engine %s: %w (%s)", e.name, err, strings.TrimSpace(stderr.String()))
	}
	if stdout.truncated {
		return nil, fmt.Errorf("engine %s: output exceeded %d bytes", e.name, maxExecOutput)
	}
	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		return nil, fmt.Errorf("engine %s: bad JSON: %w", e.name, err)
	}
	return m, nil
}

// Request runs the "request" phase and converts the returned URL/headers.
func (e *ExecEngine) Request(ctx context.Context, q engine.Query) (*engine.HTTPRequest, error) {
	m, err := e.run(ctx, map[string]any{
		"phase": "request", "query": q.Query, "pageno": q.PageNo,
		"language": q.Locale, "safesearch": q.SafeSearch,
		"time_range": q.TimeRange, "config": q.Config,
	})
	if err != nil {
		return nil, err
	}
	if errStr, _ := m["error"].(string); errStr != "" {
		return nil, fmt.Errorf("engine %s: %s", e.name, errStr)
	}
	url, _ := m["url"].(string)
	if url == "" {
		return nil, nil // skip engine for this query
	}
	req := &engine.HTTPRequest{
		Method: strOr(m["method"], "GET"), URL: url,
		Headers: map[string]string{}, Cookies: map[string]string{},
	}
	if body, ok := m["data"].(string); ok && body != "" {
		req.Body = []byte(body)
	}
	if h, ok := m["headers"].(map[string]any); ok {
		for k, v := range h {
			req.Headers[k] = fmt.Sprintf("%v", v)
		}
	}
	if c, ok := m["cookies"].(map[string]any); ok {
		for k, v := range c {
			req.Cookies[k] = fmt.Sprintf("%v", v)
		}
	}
	return req, nil
}

// Response runs the "response" phase and converts the returned result maps.
func (e *ExecEngine) Response(ctx context.Context, resp *engine.HTTPResponse) (result.EngineResults, error) {
	m, err := e.run(ctx, map[string]any{
		"phase": "response", "text": resp.Text(), "url": resp.URL,
		"status_code": resp.StatusCode, "query": resp.Query, "config": resp.Config,
	})
	if err != nil {
		return nil, err
	}
	rows, ok := m["results"].([]any)
	if !ok {
		return result.EngineResults{}, nil
	}
	res := result.EngineResults{}
	for _, row := range rows {
		rm, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := rm["suggestion"].(string); ok && s != "" {
			res = append(res, &result.Suggestion{EngineName: e.name, Value: s})
			continue
		}
		if r := result.FromMap(e.name, rm); r != nil {
			res = append(res, r)
		}
	}
	return res, nil
}

func strOr(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}
