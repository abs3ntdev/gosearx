// Command gosearx is the Phase 1 entry point. Two modes:
//
//	gosearx serve [-addr :8080] [-engines ./engines]
//	    Run the JSON API server.
//
//	gosearx search -q "query" [-engines ./engines] [-category general]
//	    Run a one-shot search and print JSON to stdout.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/searxng/gosearx/internal/ai"
	"github.com/searxng/gosearx/internal/autocomplete"
	"github.com/searxng/gosearx/internal/cache"
	"github.com/searxng/gosearx/internal/config"
	"github.com/searxng/gosearx/internal/engine"
	"github.com/searxng/gosearx/internal/engine/loader"
	"github.com/searxng/gosearx/internal/finance"
	"github.com/searxng/gosearx/internal/limiter"
	"github.com/searxng/gosearx/internal/metrics"
	"github.com/searxng/gosearx/internal/network"
	"github.com/searxng/gosearx/internal/plugin"
	_ "github.com/searxng/gosearx/internal/plugin/native" // register native plugins
	"github.com/searxng/gosearx/internal/proxy"
	"github.com/searxng/gosearx/internal/query"
	"github.com/searxng/gosearx/internal/search"
	"github.com/searxng/gosearx/internal/server"
	"github.com/searxng/gosearx/web"
)

// loadRegistry loads engine files then applies config overrides.
func loadRegistry(cfg *config.Config) *engine.Registry {
	reg, errs := loader.LoadDir(cfg.Server.EnginesDir)
	reportLoadErrors(errs)
	reportLoadErrors(loader.ApplyConfig(reg, cfg.EngineOverrides()))
	return reg
}

// loadPlugins loads native plugins plus script plugins from one or more
// directories (later dirs add to, never hide, earlier ones).
func loadPlugins(dirs ...string) *plugin.Storage {
	store, errs := plugin.LoadDirs(dirs...)
	reportLoadErrors(errs)
	return store
}

// pluginDirs builds the ordered plugin search path: the built-in "plugins"
// directory first, then any comma-separated extra dirs from -plugins / the
// GOSEARX_PLUGINS env (so Docker users can mount custom plugins without
// clobbering the built-ins).
func pluginDirs(extra string) []string {
	dirs := []string{"plugins"}
	if extra == "" {
		extra = os.Getenv("GOSEARX_PLUGINS")
	}
	for _, d := range strings.Split(extra, ",") {
		if d = strings.TrimSpace(d); d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "serve":
		cmdServe(os.Args[2:])
	case "search":
		cmdSearch(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gosearx <serve|search> [flags]")
	os.Exit(2)
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "settings.yml", "path to settings.yml")
	addr := fs.String("addr", "", "listen address (overrides config)")
	enginesDir := fs.String("engines", "", "engine dir (overrides config)")
	pluginsDir := fs.String("plugins", "", "extra plugin dir(s), comma-separated (added to built-ins)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		fatal("config: %v", err)
	}
	if *addr != "" {
		cfg.Server.Address = *addr
	}
	if *enginesDir != "" {
		cfg.Server.EnginesDir = *enginesDir
	}

	reg := loadRegistry(cfg)
	fmt.Fprintf(os.Stderr, "loaded %d engines: %v\n", len(reg.Names()), reg.Names())
	plugins := loadPlugins(pluginDirs(*pluginsDir)...)
	fmt.Fprintf(os.Stderr, "loaded %d plugins: %v\n", len(plugins.Names()), plugins.Names())

	// Valkey/Redis cache (falls back to in-memory if unset/unreachable). Used
	// for the limiter, suspended-engine tracking, AND per-engine response caching.
	store := cache.New(cfg.Valkey.URL)
	fmt.Fprintf(os.Stderr, "cache backend: %s\n", store.Backend())

	mx := metrics.New(0, 0)
	searcher := search.New(network.New()).WithPlugins(plugins).WithMetrics(mx)
	if cfg.Search.ResultCacheTTL > 0 {
		searcher = searcher.WithCache(store, cfg.Search.ResultCacheTTL)
		fmt.Fprintf(os.Stderr, "result cache enabled (ttl: %s)\n", cfg.Search.ResultCacheTTL)
	}
	if cfg.Search.CollapseDuplicates || len(cfg.Search.DomainPenalties) > 0 {
		searcher = searcher.WithEnrichment(cfg.Search.CollapseDuplicates, cfg.Search.DomainPenalties)
		fmt.Fprintf(os.Stderr, "result enrichment enabled (collapse_dups: %v, custom_penalties: %d)\n",
			cfg.Search.CollapseDuplicates, len(cfg.Search.DomainPenalties))
	}
	srv := server.New(reg, searcher).WithConfig(cfg).WithMetrics(mx)
	if cfg.Finance.Enabled {
		if fin, err := finance.NewService(cfg.Finance.Source); err != nil {
			fmt.Fprintf(os.Stderr, "warn: finance disabled: %v\n", err)
		} else {
			// Cache finance charts (5min TTL) so repeated lookups skip Yahoo.
			fin = fin.WithCache(store, 5*time.Minute)
			srv = srv.WithFinance(fin)
			fmt.Fprintf(os.Stderr, "finance enabled (source: %s, cached)\n", fin.SourceName())
		}
	}
	if cfg.AI.Enabled {
		aiSvc := ai.New(ai.Config{
			Provider: cfg.AI.Provider, BaseURL: cfg.AI.BaseURL, Model: cfg.AI.Model,
			APIKey: cfg.AI.APIKey, TopN: cfg.AI.TopN, Timeout: cfg.AI.Timeout,
		})
		srv = srv.WithAI(aiSvc, cfg.AI.Auto)
		fmt.Fprintf(os.Stderr, "AI synthesis enabled (provider: %s, model: %s, auto: %v)\n",
			cfg.AI.Provider, cfg.AI.Model, cfg.AI.Auto)
	}
	if c := autocomplete.New(cfg.Search.Autocomplete); c != nil {
		srv = srv.WithAutocomplete(c)
		fmt.Fprintf(os.Stderr, "autocomplete enabled (backend: %s)\n", cfg.Search.Autocomplete)
	}
	if cfg.Server.ImageProxy {
		srv = srv.WithImageProxy(proxy.NewImageProxy())
		fmt.Fprintln(os.Stderr, "image proxy enabled at /image_proxy")
	}
	if f := proxy.NewFaviconResolver(cfg.Search.FaviconResolver); f != nil {
		srv = srv.WithFavicon(f)
		fmt.Fprintf(os.Stderr, "favicon resolver enabled (backend: %s)\n", cfg.Search.FaviconResolver)
	}
	if assets, ok := web.Assets(); ok {
		srv = srv.WithAssets(assets)
		fmt.Fprintln(os.Stderr, "serving embedded frontend at /")
	} else {
		fmt.Fprintln(os.Stderr, "frontend not built (run: cd web && npm run build); API only")
	}

	var handler http.Handler = srv.Handler()
	if cfg.Server.Limiter.Enabled {
		handler = limiter.New(cfg.Server.Limiter, store).Middleware(handler)
		fmt.Fprintf(os.Stderr, "limiter enabled (backend: %s)\n", store.Backend())
	}
	httpSrv := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Fprintf(os.Stderr, "listening on %s\n", cfg.Server.Address)
	if err := httpSrv.ListenAndServe(); err != nil {
		fatal("server: %v", err)
	}
}

func cmdSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	q := fs.String("q", "", "search query (may include !shortcut / :category)")
	configPath := fs.String("config", "settings.yml", "path to settings.yml")
	enginesDir := fs.String("engines", "", "engine dir (overrides config)")
	pluginsDir := fs.String("plugins", "", "extra plugin dir(s), comma-separated (added to built-ins)")
	category := fs.String("category", "", "default category (overrides config)")
	timeout := fs.Duration("timeout", 15*time.Second, "overall timeout")
	_ = fs.Parse(args)

	if *q == "" {
		fatal("missing -q")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fatal("config: %v", err)
	}
	if *enginesDir != "" {
		cfg.Server.EnginesDir = *enginesDir
	}
	defCat := cfg.Search.DefaultCategory
	if *category != "" {
		defCat = *category
	}

	reg := loadRegistry(cfg)

	parsed := query.Parse(*q)
	engines := resolve(reg, parsed, defCat)
	if len(engines) == 0 {
		fatal("no engines selected for query %q", *q)
	}

	plugins := loadPlugins(pluginDirs(*pluginsDir)...)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := search.New(network.New()).WithPlugins(plugins).Search(ctx, search.Query{
		Text: parsed.Text, PageNo: 1, Locale: "all", Engines: engines,
	})
	if err != nil {
		fatal("search: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{
		"answers": resp.Answers,
		"quotes":  resp.Quotes,
		"charts":  resp.Charts,
		"results": resp.Results,
	})
	fmt.Fprintf(os.Stderr, "✓ %d results, %d answers from %d engines (%d unresponsive)\n",
		len(resp.Results), len(resp.Answers), len(resp.Timings), len(resp.Unresponsive))
	for _, u := range resp.Unresponsive {
		fmt.Fprintf(os.Stderr, "  ! %s: %s\n", u.Engine, u.Reason)
	}
}

func resolve(reg *engine.Registry, p query.Parsed, defCat string) []*engine.Registered {
	seen := map[string]bool{}
	var out []*engine.Registered
	add := func(r *engine.Registered) {
		if r != nil && !r.Meta.Disabled && !seen[r.Meta.Name] {
			seen[r.Meta.Name] = true
			out = append(out, r)
		}
	}
	for _, sc := range p.EngineShorts {
		add(reg.ByShortcut(sc))
	}
	for _, cat := range p.Categories {
		for _, r := range reg.ByCategory(cat) {
			add(r)
		}
	}
	if len(out) == 0 {
		for _, r := range reg.ByCategory(defCat) {
			add(r)
		}
	}
	return out
}

func reportLoadErrors(errs []error) {
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "warn: %v\n", e)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
