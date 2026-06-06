// Package jsruntime provides a pooled, sandboxed JavaScript runtime (goja) for
// gosearx plugins and engines. goja is pure-Go, so the binary stays CGO-free.
//
// A *goja.Runtime is not safe for concurrent use, so — exactly like the Lua
// pool — callers Acquire a runtime for the duration of a single call and Release
// it afterward. Each runtime has the compiled program's top-level code already
// executed, so any global functions (pre_search, on_result, …) are defined.
package jsruntime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/dop251/goja"
)

// Pool manages pre-initialized goja runtimes for one compiled program.
type Pool struct {
	name string
	prog *goja.Program
	mu   sync.Mutex
	free []*goja.Runtime
	max  int
}

// Compile parses a JS program once and returns a Pool that lazily builds
// sandboxed runtimes.
func Compile(name, src string) (*Pool, error) {
	prog, err := goja.Compile(name, src, true)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", name, err)
	}
	return &Pool{name: name, prog: prog, max: 8}, nil
}

// Name returns the program name.
func (p *Pool) Name() string { return p.name }

// Acquire returns a ready runtime.
func (p *Pool) Acquire() (*goja.Runtime, error) {
	p.mu.Lock()
	if n := len(p.free); n > 0 {
		rt := p.free[n-1]
		p.free = p.free[:n-1]
		p.mu.Unlock()
		return rt, nil
	}
	p.mu.Unlock()
	return p.newRuntime()
}

// Release returns a healthy runtime to the pool; unhealthy ones are discarded.
func (p *Pool) Release(rt *goja.Runtime, healthy bool) {
	if !healthy {
		return
	}
	p.mu.Lock()
	if len(p.free) < p.max {
		p.free = append(p.free, rt)
	}
	p.mu.Unlock()
}

func (p *Pool) newRuntime() (*goja.Runtime, error) {
	rt := goja.New()
	// Field name mapper so Go structs marshal with idiomatic JS/JSON names if
	// ever exposed; results travel as plain maps so this is mostly defensive.
	installSandbox(rt)
	if _, err := rt.RunProgram(p.prog); err != nil {
		return nil, fmt.Errorf("init %s: %w", p.name, err)
	}
	return rt, nil
}

// Close drops all pooled runtimes.
func (p *Pool) Close() {
	p.mu.Lock()
	p.free = nil
	p.mu.Unlock()
}

// installSandbox exposes a minimal, safe helper surface mirroring the Lua
// stdlib: url, base64, and JSON (JSON is already native in JS). No filesystem,
// no network, no process access — goja has none of those by default.
func installSandbox(rt *goja.Runtime) {
	urlObj := rt.NewObject()
	_ = urlObj.Set("encode", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		// A plain object -> querystring; anything else -> percent-encode string.
		if m, ok := arg.Export().(map[string]any); ok {
			vals := url.Values{}
			for k, v := range m {
				vals.Set(k, fmt.Sprintf("%v", v))
			}
			return rt.ToValue(vals.Encode())
		}
		return rt.ToValue(url.QueryEscape(arg.String()))
	})
	_ = urlObj.Set("escape", func(s string) string { return url.QueryEscape(s) })
	_ = urlObj.Set("unescape", func(s string) string {
		d, err := url.QueryUnescape(s)
		if err != nil {
			return s
		}
		return d
	})
	_ = rt.Set("url", urlObj)

	b64 := rt.NewObject()
	_ = b64.Set("encode", func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	})
	_ = b64.Set("decode", func(s string) string {
		d, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return ""
		}
		return string(d)
	})
	_ = rt.Set("base64", b64)
}

// MarshalReply converts a goja value into a Go map[string]any via JSON so it can
// be consumed by result.FromMap. Returns nil on failure.
func MarshalReply(rt *goja.Runtime, v goja.Value) any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	exported := v.Export()
	// Normalize through JSON so numbers become float64 and nested objects
	// become map[string]any / []any — the shapes result.FromMap expects.
	b, err := json.Marshal(exported)
	if err != nil {
		return nil
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}
