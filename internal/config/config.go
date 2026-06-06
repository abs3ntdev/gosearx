// Package config loads gosearx settings from a YAML file. The schema is a
// pragmatic subset of SearXNG's settings.yml (general/search/server/engines) so
// the format is familiar, modelling only what Phase 1 consumes.
//
// Engine entries here are *overrides/metadata*, layered on top of the engine
// definition files in the engines/ directory: an entry can enable/disable an
// engine, set its weight, timeout, shortcut, and categories.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root settings document.
type Config struct {
	General GeneralConfig  `yaml:"general"`
	Search  SearchConfig   `yaml:"search"`
	Server  ServerConfig   `yaml:"server"`
	Finance FinanceConfig  `yaml:"finance"`
	Valkey  ValkeyConfig   `yaml:"valkey"`
	AI      AIConfig       `yaml:"ai"`
	Engines []EngineConfig `yaml:"engines"`
}

// AIConfig configures the optional AI answer-synthesis feature: an LLM reads the
// top results and writes a short cited summary. Disabled by default; privacy is
// preserved by pointing it at a local Ollama by default.
type AIConfig struct {
	Enabled bool `yaml:"enabled"`
	// Provider: "ollama" (default) or "openai" (any OpenAI-compatible endpoint).
	Provider string `yaml:"provider"`
	// BaseURL of the LLM API. Ollama default: http://localhost:11434
	BaseURL string `yaml:"base_url"`
	// Model name, e.g. "llama3.2" (ollama) or "gpt-4o-mini" (openai).
	Model string `yaml:"model"`
	// APIKey for openai-compatible providers (supports ${ENV}).
	APIKey string `yaml:"api_key"`
	// TopN results fed to the model as context (default 5).
	TopN int `yaml:"top_n"`
	// Timeout for the LLM call (default 30s).
	Timeout time.Duration `yaml:"timeout"`
	// Auto runs synthesis automatically for every search when true; otherwise
	// only on explicit request (the frontend "Ask AI" affordance).
	Auto bool `yaml:"auto"`
}

// ValkeyConfig configures the Valkey/Redis backend (cache, limiter, stats).
type ValkeyConfig struct {
	URL string `yaml:"url"` // valkey://host:port/db ; empty = in-memory
}

// FinanceConfig configures the pluggable finance feature.
type FinanceConfig struct {
	Enabled bool   `yaml:"enabled"`
	Source  string `yaml:"source"` // "yahoo" | "stooq" | "google"
}

// GeneralConfig holds instance-wide options.
type GeneralConfig struct {
	InstanceName string `yaml:"instance_name"`
}

// SearchConfig holds search defaults.
type SearchConfig struct {
	DefaultCategory  string        `yaml:"default_category"`
	SafeSearch       int           `yaml:"safe_search"`
	MaxTimeout       time.Duration `yaml:"max_timeout"`
	Autocomplete     string        `yaml:"autocomplete"`     // backend name or "" (off)
	AutocompleteMin  int           `yaml:"autocomplete_min"` // min chars
	FaviconResolver  string        `yaml:"favicon_resolver"` // backend or "" (off)
	ResultCacheTTL   time.Duration `yaml:"result_cache_ttl"` // per-engine response cache TTL (0 = off)
	Formats          []string      `yaml:"formats"`          // html, json, csv, rss
	CategoriesAsTabs []string      `yaml:"categories_as_tabs"`
	// CollapseDuplicates folds near-identical results (same host + title).
	CollapseDuplicates bool `yaml:"collapse_duplicates"`
	// DomainPenalties down-ranks low-quality domains (host -> score multiplier,
	// 0..1). Empty = built-in defaults; set to disable, override, or extend.
	DomainPenalties map[string]float64 `yaml:"domain_penalties"`
}

// ServerConfig holds HTTP server options.
type ServerConfig struct {
	Address    string        `yaml:"address"`
	EnginesDir string        `yaml:"engines_dir"`
	ImageProxy bool          `yaml:"image_proxy"`
	Limiter    LimiterConfig `yaml:"limiter"`
}

// LimiterConfig configures bot/rate limiting (mirrors limiter.toml).
type LimiterConfig struct {
	Enabled        bool     `yaml:"enabled"`
	LinkToken      bool     `yaml:"link_token"`
	TrustedProxies []string `yaml:"trusted_proxies"`
	PassIP         []string `yaml:"pass_ip"`
	BlockIP        []string `yaml:"block_ip"`
}

// EngineConfig is a per-engine override keyed by name (matching an engine file).
// If Template is set, this entry *instantiates* a new engine from the named
// engine file (Lua or YAML base), enabling one engine file to back multiple
// configured engines (e.g. several MediaWiki wikis). Mirrors SearXNG's
// name/engine separation.
type EngineConfig struct {
	Name       string        `yaml:"name"`
	Template   string        `yaml:"template"`
	Disabled   bool          `yaml:"disabled"`
	Weight     float64       `yaml:"weight"`
	Timeout    time.Duration `yaml:"timeout"`
	Shortcut   string        `yaml:"shortcut"`
	Categories []string      `yaml:"categories"`
	// Config carries arbitrary per-engine settings (api_key, base_url, …)
	// passed through to Lua engines as params.config.
	Config map[string]string `yaml:"config"`
}

// Default returns a Config with sensible defaults applied.
func Default() *Config {
	return &Config{
		General: GeneralConfig{InstanceName: "gosearx"},
		Search: SearchConfig{
			DefaultCategory: "general", SafeSearch: 0, MaxTimeout: 15 * time.Second,
			AutocompleteMin: 4, Formats: []string{"html", "json"},
			ResultCacheTTL: 60 * time.Second,
		},
		Server:  ServerConfig{Address: ":8080", EnginesDir: "engines"},
		Finance: FinanceConfig{Enabled: true, Source: "yahoo"},
	}
}

// Load reads a YAML config file and merges it over defaults. A missing path
// returns the defaults (so the app runs with zero config).
func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	// Expand ${VAR} / $VAR references so secrets (e.g. GitHub tokens) can live in
	// the environment instead of plaintext in the config file.
	data = []byte(os.Expand(string(data), func(k string) string { return os.Getenv(k) }))
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	// Re-apply defaults for any field left zero by the user file.
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8080"
	}
	if cfg.Server.EnginesDir == "" {
		cfg.Server.EnginesDir = "engines"
	}
	if cfg.Search.DefaultCategory == "" {
		cfg.Search.DefaultCategory = "general"
	}
	if cfg.Search.MaxTimeout == 0 {
		cfg.Search.MaxTimeout = 15 * time.Second
	}
	return cfg, nil
}

// EngineOverrides indexes engine configs by name for quick lookup.
func (c *Config) EngineOverrides() map[string]EngineConfig {
	m := make(map[string]EngineConfig, len(c.Engines))
	for _, e := range c.Engines {
		m[e.Name] = e
	}
	return m
}
