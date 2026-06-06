# gosearx

A Go rewrite of [SearXNG](https://github.com/searxng/searxng): a privacy-respecting
metasearch engine, with first-class **Lua** engines/plugins and a modern
**React + TypeScript** frontend supporting interactive charts and rich result
types (finance, time-series, tables) beyond text and images.

> Status: **all planned phases implemented.** Working end-to-end metasearch
> (parallel engines → merge/dedup/score → JSON API → React UI), three engine
> tiers (declarative xpath/json + Lua), a Lua plugin system, a traits/locale
> subsystem, and a pluggable finance feature with interactive charts.
>
> See `docs/engines.md` and `docs/plugins.md` for authoring guides.

## Why a rewrite

The upstream Python SearXNG is excellent but, for this project's goals, lacks:
easy plugin development, interactive chart/data display types, and a finance
(Google Finance-style) integration. This port targets those directly.

## Keystone architectural decision

**Scripting is the engine substrate, not just a third-party extension point.**

SearXNG already proves the model: its generic `xpath`/`json` engines are pure
config and back a large share of its 200+ engines. Here:

- **Tier 1 — Declarative (YAML):** xpath/json engines, no code. Covers the majority.
- **Tier 2 — Lua scripts:** bespoke engines (google, wikidata, …) and plugins.
  Pure-Go [`gopher-lua`](https://github.com/yuin/gopher-lua) (no CGO → single
  static binary). Sandboxed; the host performs all I/O.
- **Tier 3 — Native Go:** the few perf-critical engines.

Adding an engine never requires recompiling Go. This makes "feature parity" and
"easy plugin development" the *same* effort.

## Scripting language: Lua

Chosen for broad familiarity, mature pure-Go VM, and a simple sync model.
Tradeoff accepted: porting the Python engines is hand-translation (Starlark would
have been near-mechanical), but the declarative tier minimizes how many engines
need Lua at all.

## Phase 0 spike — results

The spike (in `internal/engine/script` + `engines/mojeek.lua`) **validated the
two biggest risks**:

1. **XPath/HTML parity** (lxml → `antchfx/htmlquery`): a faithful Lua port of
   SearXNG's Mojeek engine — including nested/relative selectors
   (`./@href`, `../h2/a`, `..//p[@class="s"]`) and whitespace normalization —
   produces correct results against **both** a fixture and the **live** Mojeek site.
2. **Concurrency**: engines run one-goroutine-each. A per-engine **LState pool**
   with `context` deadlines passes 50-way concurrent stress under `-race`, and a
   runaway script is correctly killed by its context timeout.

## Run it (Phase 1)

```sh
# one-shot search across enabled engines (parallel, merged, scored)
go run ./cmd/gosearx search -q "go programming language"

# select a single engine with a !bang
go run ./cmd/gosearx search -q "!mj golang"

# build the frontend (embedded into the binary), then serve UI + API
cd web && npm install && npm run build && cd ..
go run ./cmd/gosearx serve            # reads settings.yml
#   GET /                      -> React UI
#   GET /api/search?q=...&pageno=1&safesearch=0&time_range=
#   GET /api/engines
#   GET /healthz

# frontend dev server with API proxy to :8080
cd web && npm run dev

go test -race ./...
```

## Configuration (`settings.yml`)

A familiar subset of SearXNG's format. Engine *definitions* live in `engines/`;
this file configures the instance and *overrides* engine metadata by name:

```yaml
server: { address: ":8080", engines_dir: engines }
search: { default_category: general, max_timeout: 15s }
engines:
  - name: mojeek
    weight: 1.0
    timeout: 3s
  - name: mojeek-declarative
    disabled: true
```

## Engine contract (Lua)

```lua
function request(query, params)
  params.url = "https://example.com/search?" .. url.encode({ q = query })
  return params                      -- host executes the request; engines are pure
end

function response(resp)
  local results, dom = {}, html.parse(resp.text)
  for _, r in ipairs(xpath.list(dom, '//li/a')) do
    results[#results+1] = { url = xpath.text(r, './@href'), title = xpath.text(r, '.') }
  end
  return results
end
```

Sandboxed API exposed to scripts: `html.parse`, `xpath.{list,first,text,attr,url}`,
`url.{encode,escape}`, `json.{decode,encode}`, `base64.{encode,decode}`.
Denied: `os`, `io`, `require`, `load*`, filesystem/network/process access.

## Layout

```
cmd/gosearx/         CLI: `serve` (API+UI) and `search` (one-shot)
internal/
  result/            typed result union (main/image/answer/quote/chart/…)
  lua/               shared sandboxed Lua env: pool, stdlib, html/xpath bindings
  engine/            engine contract (Request/Response, no dict mutation)
    script/          Lua engine tier (uses internal/lua)
    declarative/     yaml xpath + json engine tiers (zero code)
    loader/          dispatches .yaml -> declarative, .lua -> script
  htmlx/             shared lxml-equivalent HTML/XPath helpers
  plugin/            Lua plugin tier: pre_search/on_result/post_search + storage
  traits/            locale matching (engine language/region best-fit)
  finance/           pluggable market-data sources (yahoo, stooq) + detection
  query/             query parser (!bang, :category)
  network/           HTTP fetch layer (context timeouts, Fetcher iface)
  search/            orchestrator: parallel engines + merge/dedup/score + plugins
  server/            JSON API (/api/search, /api/engines, /healthz) + SPA
  config/            settings.yml loader (+ per-engine overrides, finance)
engines/   mojeek.{lua,yaml} (same engine both tiers), wikipedia.lua
plugins/   calculator.lua, tracker_remover.lua, hash.lua
web/       React + TS app; result-type registry; embedded via embed.FS
  src/results/registry.tsx   maps result type -> component (extensibility point)
  src/results/ChartCard.tsx  interactive finance chart (lightweight-charts)
docs/      engines.md, plugins.md  (authoring guides)
```

## Roadmap

- **Phase 0 — Spikes** ✅ Lua runtime + XPath parity + concurrency.
- **Phase 1 — Walking skeleton** ✅ query parser (`!bang`/`:category`),
  parallel orchestrator (goroutines + context + per-engine timeout isolation),
  result merge/dedup/score/group, declarative **xpath + json** engine tiers,
  engine registry + loader, **settings.yml** config (enable/disable, weights,
  timeouts), JSON API (`/api/search`, `/api/engines`, `/healthz`), and a
  **React + TS frontend** with a result-type registry, embedded into the binary
  via `embed.FS` (single self-contained executable).
- **Phase 2 — Scripting substrate + plugins** ✅ shared sandboxed Lua env,
  plugin hooks (`pre_search`/`on_result`/`post_search`), keyword gating, built-in
  plugins (calculator, tracker remover, hash); instant-answer mechanism.
- **Phase 3 — Parity subsystems** ✅ traits/locale best-fit matching, image
  result type, additional engines across both tiers.
- **Phase 4 — Finance + rich frontend** ✅ pluggable finance datasource
  (yahoo/stooq), ticker detection, `quote`/`chart` result types, interactive
  candlestick charts (lightweight-charts), answer/image components.
- **Phase 5 — Polish** ✅ engine/plugin authoring docs, Makefile, single-binary
  self-host (embedded assets via `embed.FS`).

### Production features (ported from the user's SearXNG instance)

- **Output formats:** `?format=json|csv|rss` on `/api/search`.
- **Autocomplete:** `/api/autocomplete` (backends: google, duckduckgo, brave).
- **Image proxy:** `/image_proxy?url=…` (privacy: thumbnails fetched server-side).
- **Favicon resolver:** `/favicon_proxy?host=…` (duckduckgo/google/allesedv/yandex).
- **Limiter:** per-IP token bucket, trusted-proxy XFF resolution, pass/block
  IP lists (mirrors the old `limiter.toml`, Pangolin/Traefik setup).
- **Per-engine config + secrets:** `engines[].config` (e.g. Brave `api_key`).
- **Template engines:** one engine file backs many configured engines via
  `engines[].template` (e.g. StackExchange sites, MediaWiki wikis).

### Engines ported so far (~25)

braveapi (API key), google, duckduckgo, github, stackoverflow/askubuntu/superuser,
mdn, mankier, hoogle, packagist, wiby, bitbucket, pub.dev, rubygems, etymonline,
habrahabr, lobste.rs, mediawiki (arch/gentoo/wiktionary), wikipedia, mojeek.

### Plugins ported

calculator, unit_converter, hostnames, tracker_remover, hash, and the user's
custom **eth_checksum** (native Go: EIP-55 + ENS resolution).

### Not yet done (future work)

Bulk-porting the remaining ~195 SearXNG engines (the substrate is ready — this is
porting-by-data: declarative YAML or short Lua), preferences UI, valkey cache,
metrics/OpenMetrics, and full i18n. The architecture supports all of these.

## License

Derives from SearXNG (AGPL-3.0-or-later); this project is AGPL-3.0-or-later.
