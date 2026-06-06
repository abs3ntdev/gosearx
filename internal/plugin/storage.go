package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/searxng/gosearx/internal/result"
)

// Storage holds the loaded plugins and runs their hooks. Port of SearXNG's
// PluginStorage.
type Storage struct {
	plugins []Plugin
}

// NewStorage returns an empty Storage.
func NewStorage() *Storage { return &Storage{} }

// Add registers a plugin.
func (s *Storage) Add(p Plugin) { s.plugins = append(s.plugins, p) }

// Names lists registered plugin names.
func (s *Storage) Names() []string {
	out := make([]string, 0, len(s.plugins))
	for _, p := range s.plugins {
		out = append(out, p.Name())
	}
	return out
}

// nativeFactories holds in-process native Go plugins registered via Register.
var nativeFactories []func() Plugin

// Register adds a native Go plugin factory. Called from plugin init() functions.
func Register(factory func() Plugin) {
	nativeFactories = append(nativeFactories, factory)
}

// LoadDir loads native plugins then every script plugin in dir.
func LoadDir(dir string) (*Storage, []error) {
	return LoadDirs(dir)
}

// LoadDirs loads the compiled-in native plugins once, then every script plugin
// found across the given directories (in order). This lets a deployment ship
// built-in plugins in one directory and mount user/custom plugins in another
// (e.g. a Docker bind-mount) without one hiding the other. A missing directory
// is skipped silently.
func LoadDirs(dirs ...string) (*Storage, []error) {
	s := NewStorage()
	// Native (compiled-in) plugins first — registered exactly once.
	for _, f := range nativeFactories {
		s.Add(f())
	}
	var errs []error
	seen := map[string]bool{}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		errs = append(errs, s.loadDirInto(dir, seen)...)
	}
	return s, errs
}

// loadDirInto scans one directory for script plugins, skipping plugin names
// already loaded (so an earlier directory's plugin can be overridden by name,
// and the same name isn't loaded twice).
func (s *Storage) loadDirInto(dir string, seen map[string]bool) []error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // a missing plugins dir is fine
		}
		return []error{fmt.Errorf("read plugins dir %q: %w", dir, err)}
	}
	var errs []error
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		// .lua (in-process Lua), .mjs (in-process JS), or any exec-able script.
		if ext == ".lua" || ext == ".mjs" || IsExecExtension(ext) {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		p, err := loadPluginFile(f)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f, err))
			continue
		}
		if p == nil {
			continue
		}
		if seen[p.Name()] {
			continue // already loaded (earlier dir or a native of the same name)
		}
		seen[p.Name()] = true
		s.Add(p)
	}
	return errs
}

// loadPluginFile compiles a single plugin file based on its extension:
//   - .lua          -> in-process Lua (gopher-lua)
//   - .mjs          -> in-process JavaScript (goja)
//   - .sh/.py/.rb/… -> external interpreter via the exec backend
//   - .plugin       -> arbitrary chmod+x executable via the exec backend
//
// Note: .js is treated as an EXEC plugin (node) so it can use the full Node
// ecosystem; use .mjs for the sandboxed in-process goja backend.
func loadPluginFile(path string) (Plugin, error) {
	ext := strings.ToLower(filepath.Ext(path))
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	switch {
	case ext == ".lua":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return CompileLua(name, string(data))
	case ext == ".mjs":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return CompileJS(name, string(data))
	case IsExecExtension(ext):
		return CompileExec(path, 0)
	}
	return nil, nil
}

// PreSearch runs all applicable pre_search hooks. Returns false if any plugin
// aborts the search.
func (s *Storage) PreSearch(sc *SearchContext) bool {
	for _, p := range s.plugins {
		if !MatchKeyword(p.Keywords(), sc.Query) {
			continue
		}
		allow, err := p.PreSearch(sc)
		if err != nil {
			continue // a failing plugin shouldn't break search
		}
		if !allow {
			return false
		}
	}
	return true
}

// OnResult runs on_result for every plugin against r. Returns false if any
// plugin drops the result.
func (s *Storage) OnResult(r *result.MainResult) bool {
	for _, p := range s.plugins {
		if len(p.Keywords()) > 0 {
			continue // keyword plugins don't filter results
		}
		keep, err := p.OnResult(r)
		if err != nil {
			continue
		}
		if !keep {
			return false
		}
	}
	return true
}

// PostSearch runs all applicable post_search hooks, collecting added results.
func (s *Storage) PostSearch(sc *SearchContext) result.EngineResults {
	var added result.EngineResults
	for _, p := range s.plugins {
		if !MatchKeyword(p.Keywords(), sc.Query) {
			continue
		}
		res, err := p.PostSearch(sc)
		if err != nil {
			continue
		}
		added = append(added, res...)
	}
	return added
}
