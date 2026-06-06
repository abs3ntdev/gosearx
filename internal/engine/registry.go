package engine

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Meta holds the configuration/metadata for a registered engine that the
// orchestrator needs (independent of the engine's tier). Mirrors the per-engine
// settings SearXNG stores as module attributes.
type Meta struct {
	Name       string
	Shortcut   string
	Categories []string
	Timeout    time.Duration
	Disabled   bool
	Weight     float64 // scoring weight (SearXNG default 1.0)
	// Config carries per-engine settings (api_key, base_url, …) for Lua engines.
	Config map[string]string
}

// Registered pairs an Engine implementation with its Meta.
type Registered struct {
	Engine Engine
	Meta   Meta
}

// Registry is the set of loaded engines, indexed by name, shortcut, and category.
type Registry struct {
	mu      sync.RWMutex
	byName  map[string]*Registered
	byShort map[string]*Registered
	byCat   map[string][]*Registered
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		byName:  map[string]*Registered{},
		byShort: map[string]*Registered{},
		byCat:   map[string][]*Registered{},
	}
}

// Register adds an engine. Later registrations with the same name replace.
func (r *Registry) Register(e Engine, m Meta) {
	if m.Name == "" {
		m.Name = e.Name()
	}
	if m.Timeout == 0 {
		m.Timeout = 3 * time.Second
	}
	if m.Weight == 0 {
		m.Weight = 1.0
	}
	if len(m.Categories) == 0 {
		m.Categories = []string{"general"}
	}
	reg := &Registered{Engine: e, Meta: m}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.byName[m.Name] = reg
	if m.Shortcut != "" {
		r.byShort[m.Shortcut] = reg
	}
	for _, c := range m.Categories {
		r.byCat[c] = append(r.byCat[c], reg)
	}
}

// Override mutates an already-registered engine's metadata. Zero-valued fields
// in the override are ignored (except Disabled, which is always applied). It
// reindexes shortcut/category if those change. Returns false if not found.
func (r *Registry) Override(name string, apply func(*Meta)) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	reg, ok := r.byName[name]
	if !ok {
		return false
	}
	oldShort := reg.Meta.Shortcut
	oldCats := reg.Meta.Categories
	apply(&reg.Meta)

	if reg.Meta.Shortcut != oldShort {
		if oldShort != "" {
			delete(r.byShort, oldShort)
		}
		if reg.Meta.Shortcut != "" {
			r.byShort[reg.Meta.Shortcut] = reg
		}
	}
	if !sameStrings(reg.Meta.Categories, oldCats) {
		for _, c := range oldCats {
			r.byCat[c] = removeReg(r.byCat[c], reg)
		}
		for _, c := range reg.Meta.Categories {
			r.byCat[c] = append(r.byCat[c], reg)
		}
	}
	return true
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func removeReg(s []*Registered, r *Registered) []*Registered {
	out := s[:0]
	for _, x := range s {
		if x != r {
			out = append(out, x)
		}
	}
	return out
}

// ByName returns a registered engine or nil.
func (r *Registry) ByName(name string) *Registered {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[name]
}

// Instantiate registers a new engine that reuses an existing engine's
// implementation (the template) under a new name + Meta. Used so one engine
// file can back several configured engines (e.g. multiple MediaWiki wikis).
// Lua/declarative engines are stateless w.r.t. name — per-instance behavior
// comes from Meta.Config passed at query time. Returns false if template missing.
func (r *Registry) Instantiate(templateName string, m Meta) bool {
	r.mu.RLock()
	tmpl, ok := r.byName[templateName]
	r.mu.RUnlock()
	if !ok {
		return false
	}
	r.Register(tmpl.Engine, m)
	return true
}

// ByShortcut resolves a !bang shortcut to an engine or nil.
func (r *Registry) ByShortcut(sc string) *Registered {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byShort[sc]
}

// ByCategory returns enabled engines in a category.
func (r *Registry) ByCategory(cat string) []*Registered {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Registered
	for _, e := range r.byCat[cat] {
		if !e.Meta.Disabled {
			out = append(out, e)
		}
	}
	return out
}

// Names returns all registered engine names, sorted.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byName))
	for n := range r.byName {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Categories returns all known category names, sorted.
func (r *Registry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byCat))
	for c := range r.byCat {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// EngineFile describes a discovered engine file for loading.
type EngineFile struct {
	Path string
	Ext  string // ".yaml" or ".lua"
}

// DiscoverEngineFiles lists engine definition files in dir.
func DiscoverEngineFiles(dir string) ([]EngineFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []EngineFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		switch ext {
		case ".yaml", ".yml", ".lua", ".mjs",
			".sh", ".bash", ".py", ".rb", ".pl", ".engine":
			out = append(out, EngineFile{Path: filepath.Join(dir, e.Name()), Ext: ext})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
