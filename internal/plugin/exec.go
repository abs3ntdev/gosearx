package plugin

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

	"github.com/searxng/gosearx/internal/result"
)

// ExecPlugin is a Plugin backed by an external executable (bash, python, node,
// ruby, a compiled binary — anything). The host speaks a simple JSON protocol
// over stdin/stdout so plugin authors can use whatever language fits the task.
//
// This keeps the gosearx binary itself pure-Go / CGO-free: the interpreters are
// the user's, invoked as subprocesses, and entirely optional.
//
// # PROTOCOL
//
// For each hook the host writes one JSON object to the script's stdin:
//
//	{"hook":"pre_search","query":"...","client_ip":"...","user_agent":"..."}
//	{"hook":"on_result","result":{"title","url","content","engine"}}
//	{"hook":"post_search","query":"...","client_ip":"...","user_agent":"..."}
//
// The script must write one JSON object to stdout and exit:
//
//	pre_search   -> {"allow":true}            (false aborts the search)
//	on_result    -> {"keep":true,"result":{"title","url","content"}}
//	post_search  -> {"results":[ {<result map>}, ... ]}
//
// Result maps use the same schema as Lua/native plugins (consumed by
// result.FromMap): e.g. {"type":"answer","answer":"42"} or a web result
// {"title","url","content"}.
//
// METADATA / KEYWORDS
//
// Header comment lines near the top of the script declare metadata, e.g.:
//
//	# @keywords: weather, forecast
//	# @name: my_weather
//
// Comment markers understood: '#', '//', '--'. Keywords gate the plugin to
// queries whose first term matches (same rule as Lua plugins).
type ExecPlugin struct {
	name     string
	path     string   // absolute path to the executable
	interp   []string // optional interpreter prefix (e.g. ["bash"], ["python3"])
	keywords []string
	timeout  time.Duration
}

// maxExecOutput bounds how much a plugin may write to stdout (defense against a
// runaway script OOMing the host). 4 MiB is far more than any result payload.
const maxExecOutput = 4 << 20

// cappedBuffer is an io.Writer that stops accumulating after limit bytes and
// records that truncation occurred.
type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.buf.Len() >= c.limit {
		c.truncated = true
		return len(p), nil // pretend success; we just drop the overflow
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

// execHook is the request envelope sent to the script.
type execHook struct {
	Hook      string         `json:"hook"`
	Query     string         `json:"query,omitempty"`
	ClientIP  string         `json:"client_ip,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
}

// execReply is the response envelope returned by the script.
type execReply struct {
	Allow   *bool            `json:"allow,omitempty"`
	Keep    *bool            `json:"keep,omitempty"`
	Result  map[string]any   `json:"result,omitempty"`
	Results []map[string]any `json:"results,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// execExtensions maps file extensions to their interpreter command. A file with
// no recognized extension is executed directly (must be chmod +x).
var execExtensions = map[string][]string{
	".sh":   {"bash"},
	".bash": {"bash"},
	".py":   {"python3"},
	".js":   {"node"}, // node-as-exec; the in-process goja backend handles .mjs
	".rb":   {"ruby"},
	".pl":   {"perl"},
}

// IsExecExtension reports whether the loader should treat a file as an exec
// plugin. We accept known interpreter extensions plus the ".plugin" marker for
// arbitrary chmod+x executables.
func IsExecExtension(ext string) bool {
	if ext == ".plugin" {
		return true
	}
	_, ok := execExtensions[strings.ToLower(ext)]
	return ok
}

// CompileExec prepares an exec-backed plugin from a script path. It does not run
// the script; it only reads the header for metadata.
func CompileExec(path string, defaultTimeout time.Duration) (*ExecPlugin, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(abs))
	p := &ExecPlugin{
		name:    strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)),
		path:    abs,
		interp:  execExtensions[ext], // nil for ".plugin" => direct exec
		timeout: defaultTimeout,
	}
	if p.timeout <= 0 {
		p.timeout = 5 * time.Second
	}
	parseExecHeader(string(data), p)
	return p, nil
}

func (p *ExecPlugin) Name() string       { return p.name }
func (p *ExecPlugin) Keywords() []string { return p.keywords }

// run invokes the script with the given hook envelope and decodes the reply.
func (p *ExecPlugin) run(h execHook) (*execReply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	args := append([]string{}, p.interp...)
	var cmd *exec.Cmd
	if len(args) > 0 {
		cmd = exec.CommandContext(ctx, args[0], append(args[1:], p.path)...)
	} else {
		cmd = exec.CommandContext(ctx, p.path)
	}
	in, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}
	cmd.Stdin = bytes.NewReader(in)
	// Minimal, hermetic environment: no inherited secrets leak to plugins.
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "GOSEARX_PLUGIN=1"}
	// Run the child in its own process group and, on timeout/cancel, kill the
	// WHOLE group — otherwise a hung grandchild (e.g. `sleep`) would keep the
	// call blocked until it finishes despite the deadline.
	cmd.SysProcAttr = detachedSysProcAttr()
	cmd.Cancel = func() error { killProcessGroup(cmd); return nil }
	cmd.WaitDelay = time.Second

	// Cap stdout/stderr so a runaway plugin can't OOM the host.
	stdout := &cappedBuffer{limit: maxExecOutput}
	stderr := &cappedBuffer{limit: 8 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec plugin %s: %w (%s)", p.name, err, strings.TrimSpace(stderr.String()))
	}
	if stdout.truncated {
		return nil, fmt.Errorf("exec plugin %s: output exceeded %d bytes", p.name, maxExecOutput)
	}
	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return &execReply{}, nil
	}
	var reply execReply
	if err := json.Unmarshal(out, &reply); err != nil {
		return nil, fmt.Errorf("exec plugin %s: bad JSON reply: %w", p.name, err)
	}
	if reply.Error != "" {
		return nil, fmt.Errorf("exec plugin %s: %s", p.name, reply.Error)
	}
	return &reply, nil
}

// PreSearch runs the pre_search hook. Missing/empty reply = allow.
func (p *ExecPlugin) PreSearch(sc *SearchContext) (bool, error) {
	reply, err := p.run(execHook{
		Hook: "pre_search", Query: sc.Query,
		ClientIP: sc.ClientIP, UserAgent: sc.UserAgent,
	})
	if err != nil {
		return true, err
	}
	if reply.Allow != nil {
		return *reply.Allow, nil
	}
	return true, nil
}

// OnResult runs the on_result hook; the returned result map (if any) mutates
// title/url/content in place.
func (p *ExecPlugin) OnResult(r *result.MainResult) (bool, error) {
	reply, err := p.run(execHook{
		Hook: "on_result",
		Result: map[string]any{
			"title": r.Title, "url": r.URL,
			"content": r.Content, "engine": r.EngineName,
		},
	})
	if err != nil {
		return true, err
	}
	if reply.Keep != nil && !*reply.Keep {
		return false, nil
	}
	if reply.Result != nil {
		if v, ok := reply.Result["title"].(string); ok {
			r.Title = v
		}
		if v, ok := reply.Result["url"].(string); ok {
			r.URL = v
		}
		if v, ok := reply.Result["content"].(string); ok {
			r.Content = v
		}
	}
	return true, nil
}

// PostSearch runs the post_search hook and converts returned result maps.
func (p *ExecPlugin) PostSearch(sc *SearchContext) (result.EngineResults, error) {
	reply, err := p.run(execHook{
		Hook: "post_search", Query: sc.Query,
		ClientIP: sc.ClientIP, UserAgent: sc.UserAgent,
	})
	if err != nil {
		return nil, err
	}
	var out result.EngineResults
	for _, m := range reply.Results {
		if r := result.FromMap(p.name, m); r != nil {
			out = append(out, r)
		}
	}
	return out, nil
}

// parseExecHeader scans leading comment lines for `@key: value` directives.
func parseExecHeader(src string, p *ExecPlugin) {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip a leading comment marker; stop at the first real code line.
		marker := ""
		for _, m := range []string{"#", "//", "--"} {
			if strings.HasPrefix(line, m) {
				marker = m
				break
			}
		}
		if marker == "" {
			// Allow a shebang line then keep scanning.
			if strings.HasPrefix(line, "#!") {
				continue
			}
			break
		}
		body := strings.TrimSpace(strings.TrimPrefix(line, marker))
		if !strings.HasPrefix(body, "@") {
			continue
		}
		kv := strings.SplitN(strings.TrimPrefix(body, "@"), ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		switch key {
		case "name":
			if val != "" {
				p.name = val
			}
		case "keywords":
			for _, kw := range strings.Split(val, ",") {
				if kw = strings.TrimSpace(kw); kw != "" {
					p.keywords = append(p.keywords, kw)
				}
			}
		case "timeout":
			if d, err := time.ParseDuration(val); err == nil {
				p.timeout = d
			}
		}
	}
}
