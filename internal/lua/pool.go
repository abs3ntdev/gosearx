package lua

import (
	"fmt"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

// Pool manages a set of pre-initialized, sandboxed LStates for one compiled
// script. *lua.LState is not goroutine-safe, so callers Acquire a state for the
// duration of a call and Release it afterward (or discard if it errored).
type Pool struct {
	name     string
	proto    *lua.FunctionProto
	mu       sync.Mutex
	free     []*lua.LState
	max      int
	withHTTP bool // expose the controlled http.get capability (engines only)
}

// Compile parses a script once and returns a Pool that lazily builds states.
// The resulting states are network-free (suitable for plugins).
func Compile(name, src string) (*Pool, error) {
	return compile(name, src, false)
}

// CompileWithHTTP is like Compile but the states expose the controlled
// http.get capability for engines that need follow-up requests.
func CompileWithHTTP(name, src string) (*Pool, error) {
	return compile(name, src, true)
}

func compile(name, src string, withHTTP bool) (*Pool, error) {
	chunk, err := parse.Parse(strings.NewReader(src), name)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", name, err)
	}
	proto, err := lua.Compile(chunk, name)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", name, err)
	}
	return &Pool{name: name, proto: proto, max: 8, withHTTP: withHTTP}, nil
}

// Name returns the script name.
func (p *Pool) Name() string { return p.name }

// Acquire returns a ready LState (the script's top-level chunk has run, defining
// its global functions).
func (p *Pool) Acquire() (*lua.LState, error) {
	p.mu.Lock()
	if n := len(p.free); n > 0 {
		L := p.free[n-1]
		p.free = p.free[:n-1]
		p.mu.Unlock()
		return L, nil
	}
	p.mu.Unlock()
	return p.newState()
}

// Release returns a healthy state to the pool; unhealthy states are closed.
func (p *Pool) Release(L *lua.LState, healthy bool) {
	if !healthy {
		L.Close()
		return
	}
	L.SetTop(0)
	p.mu.Lock()
	if len(p.free) < p.max {
		p.free = append(p.free, L)
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()
	L.Close()
}

func (p *Pool) newState() (*lua.LState, error) {
	L := NewState()
	if p.withHTTP {
		RegisterHTTPAPI(L)
	}
	lfunc := L.NewFunctionFromProto(p.proto)
	L.Push(lfunc)
	if err := L.PCall(0, lua.MultRet, nil); err != nil {
		L.Close()
		return nil, fmt.Errorf("init %s: %w", p.name, err)
	}
	return L, nil
}

// Close releases all pooled states.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, L := range p.free {
		L.Close()
	}
	p.free = nil
}
