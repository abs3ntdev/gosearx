// Package loader builds an engine.Registry from a directory of engine files,
// dispatching by extension: .yaml/.yml -> declarative tier, .lua -> script tier.
// Kept separate from package engine to avoid an import cycle.
package loader

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/config"
	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/engine/declarative"
	"github.com/searxng/gosearx/internal/engine/execengine"
	"github.com/searxng/gosearx/internal/engine/jsscript"
	"github.com/searxng/gosearx/internal/engine/script"
	"gopkg.in/yaml.v3"
)

// LoadDir discovers and loads every engine file in dir into a new registry.
// Errors for individual engines are collected and returned together so one bad
// engine doesn't abort the whole load.
func LoadDir(dir string) (*engine.Registry, []error) {
	reg := engine.NewRegistry()
	files, err := engine.DiscoverEngineFiles(dir)
	if err != nil {
		return reg, []error{fmt.Errorf("discover engines: %w", err)}
	}
	var errs []error
	for _, f := range files {
		if err := loadFile(reg, f); err != nil {
			errs = append(errs, err)
		}
	}
	return reg, errs
}

// ApplyConfig layers per-engine settings overrides onto a loaded registry:
// enable/disable, weight, timeout, shortcut, categories. Unknown engine names
// in the config are returned as warnings.
func ApplyConfig(reg *engine.Registry, overrides map[string]config.EngineConfig) []error {
	var warns []error
	// First pass: instantiate template-based engines so subsequent overrides
	// (and the engine itself) exist under their configured name.
	for name, oc := range overrides {
		if oc.Template == "" {
			continue
		}
		meta := engine.Meta{
			Name:       name,
			Shortcut:   oc.Shortcut,
			Categories: oc.Categories,
			Timeout:    oc.Timeout,
			Weight:     oc.Weight,
			Disabled:   oc.Disabled,
			Config:     oc.Config,
		}
		if !reg.Instantiate(oc.Template, meta) {
			warns = append(warns, fmt.Errorf("config: engine %q references unknown template %q", name, oc.Template))
		}
	}
	// Second pass: apply overrides to existing (incl. just-instantiated) engines.
	for name, oc := range overrides {
		if oc.Template != "" {
			continue // already created with full meta above
		}
		ok := reg.Override(name, func(m *engine.Meta) {
			m.Disabled = oc.Disabled
			if oc.Weight != 0 {
				m.Weight = oc.Weight
			}
			if oc.Timeout != 0 {
				m.Timeout = oc.Timeout
			}
			if oc.Shortcut != "" {
				m.Shortcut = oc.Shortcut
			}
			if len(oc.Categories) > 0 {
				m.Categories = oc.Categories
			}
			if len(oc.Config) > 0 {
				m.Config = oc.Config
			}
		})
		if !ok {
			warns = append(warns, fmt.Errorf("config: unknown engine %q (no matching engine file)", name))
		}
	}
	return warns
}

func loadFile(reg *engine.Registry, f engine.EngineFile) error {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return fmt.Errorf("%s: %w", f.Path, err)
	}
	switch f.Ext {
	case ".yaml", ".yml":
		return loadYAML(reg, f.Path, data)
	case ".lua":
		return loadLua(reg, f.Path, data)
	case ".mjs":
		return loadJS(reg, f.Path, data)
	case ".sh", ".bash", ".py", ".rb", ".pl", ".engine":
		return loadExec(reg, f.Path, data)
	default:
		return fmt.Errorf("%s: unsupported engine ext %q", f.Path, f.Ext)
	}
}

// loadJS compiles a goja (.mjs) engine. Metadata uses `// @key: value` headers.
func loadJS(reg *engine.Registry, path string, data []byte) error {
	name := baseName(path)
	eng, err := jsscript.Compile(name, string(data))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	meta := engine.Meta{Name: name}
	parseHeaderDirectives(string(data), []string{"//", "--", "#"}, &meta)
	reg.Register(eng, meta)
	return nil
}

// loadExec registers an external-script engine. Metadata uses `#`/`//`/`--`
// `@key: value` header comments.
func loadExec(reg *engine.Registry, path string, data []byte) error {
	name := baseName(path)
	eng, err := execengine.Compile(name, path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	meta := engine.Meta{Name: name}
	parseHeaderDirectives(string(data), []string{"#", "//", "--"}, &meta)
	reg.Register(eng, meta)
	return nil
}

// parseHeaderDirectives reads "@key: value" lines from leading comments using
// any of the provided markers. Shared by JS and exec engine loaders.
func parseHeaderDirectives(src string, markers []string, m *engine.Meta) {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#!") {
			continue
		}
		marker := ""
		for _, mk := range markers {
			if strings.HasPrefix(line, mk) {
				marker = mk
				break
			}
		}
		if marker == "" {
			break // first non-comment line
		}
		body := strings.TrimSpace(strings.TrimPrefix(line, marker))
		if !strings.HasPrefix(body, "@") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(body, "@"), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch key {
		case "shortcut":
			m.Shortcut = val
		case "categories":
			var cats []string
			for _, c := range strings.Split(val, ",") {
				if c = strings.TrimSpace(c); c != "" {
					cats = append(cats, c)
				}
			}
			m.Categories = cats
		case "timeout":
			if d, err := time.ParseDuration(val); err == nil {
				m.Timeout = d
			}
		}
	}
}

func loadYAML(reg *engine.Registry, path string, data []byte) error {
	// Peek at the engine type discriminator.
	var head struct {
		Engine string `yaml:"engine"`
	}
	if err := yaml.Unmarshal(data, &head); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	switch head.Engine {
	case "xpath", "":
		var cfg declarative.XPathConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		eng, err := declarative.NewXPath(cfg)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		reg.Register(eng, engine.Meta{
			Name:       cfg.Name,
			Shortcut:   cfg.Shortcut,
			Categories: cfg.Categories,
		})
		return nil
	case "json", "json_engine":
		var cfg declarative.JSONConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		eng, err := declarative.NewJSON(cfg)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		reg.Register(eng, engine.Meta{
			Name:       cfg.Name,
			Shortcut:   cfg.Shortcut,
			Categories: cfg.Categories,
		})
		return nil
	default:
		return fmt.Errorf("%s: unknown declarative engine type %q", path, head.Engine)
	}
}

func loadLua(reg *engine.Registry, path string, data []byte) error {
	name := baseName(path)
	eng, err := script.Compile(name, string(data))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	// Lua engines may declare metadata via a leading comment directive
	// (-- @shortcut: x, -- @categories: a,b). Parse a minimal subset.
	meta := engine.Meta{Name: name}
	parseLuaDirectives(string(data), &meta)
	reg.Register(eng, meta)
	return nil
}

// parseLuaDirectives reads simple "-- @key: value" header comments for metadata.
func parseLuaDirectives(src string, m *engine.Meta) {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-- @") {
			if strings.HasPrefix(line, "--") || line == "" {
				continue
			}
			break // stop at first non-comment line
		}
		kv := strings.TrimPrefix(line, "-- @")
		parts := strings.SplitN(kv, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "shortcut":
			m.Shortcut = val
		case "categories":
			var cats []string
			for _, c := range strings.Split(val, ",") {
				if c = strings.TrimSpace(c); c != "" {
					cats = append(cats, c)
				}
			}
			m.Categories = cats
		case "timeout":
			if d, err := time.ParseDuration(val); err == nil {
				m.Timeout = d
			}
		}
	}
}

func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			p = p[i+1:]
			break
		}
	}
	if i := strings.LastIndexByte(p, '.'); i >= 0 {
		p = p[:i]
	}
	return p
}
