# gosearx

A privacy-respecting metasearch engine written in Go, inspired by
[SearXNG](https://github.com/searxng/searxng). It aggregates results from ~70
engines in parallel, scriptable in **Lua, JavaScript, or any executable**, with a
**React + TypeScript** frontend, optional **AI answer synthesis**, first-class
**GitHub** search, and an integrated **finance** feature with interactive charts.

Ships as a single self-contained binary (the frontend is embedded) — no CGO, no
runtime dependencies.

## Highlights

- **~70 engines** across general, images, videos, news, IT, science, files,
  social, music, and map categories — including google, bing, duckduckgo, brave,
  startpage, mojeek, wikipedia/wikidata, arxiv, and more.
- **Multi-language engines & plugins.** Write integrations in:
  - **Declarative YAML** (xpath/json) — no code,
  - **Lua** (in-process, sandboxed, pure-Go `gopher-lua`),
  - **JavaScript** (in-process, sandboxed, pure-Go `goja`, `.mjs`),
  - **Exec scripts** (bash, python, ruby, node, … via a JSON stdio protocol),
  - **Native Go** for the perf-critical few.
- **AI answer synthesis** (optional): an LLM reads the top results and writes a
  cited summary. Pluggable provider — local **Ollama** by default (private), or
  any OpenAI-compatible endpoint.
- **First-class GitHub search:** repos, code, issues/PRs, users, topics, commits,
  discussions with intent routing, unified ranking, rich cards, README preview,
  and copy buttons.
- **Finance:** stock/crypto quotes and interactive candlestick charts
  (lightweight-charts) with range selection; pluggable datasources (yahoo/stooq).
- **Instant answers:** calculator, unit/currency conversion, dictionary, IP/DNS,
  hashes, base/timestamp/color conversion, password/uuid/lorem/roman generators,
  and more.
- **Result quality:** cross-engine dedup/scoring, near-duplicate collapsing, and
  configurable SEO/content-farm down-ranking.
- **Search filters:** time range, language, safesearch, file type, and
  site include/exclude.
- **Production features:** Valkey/Redis caching (in-memory fallback), per-IP
  limiter, cookie preferences, OpenSearch, metrics/OpenMetrics, autocomplete,
  image/favicon proxy, and `json`/`csv`/`rss` output.
- **Modern UI:** React + TypeScript SPA with a Catppuccin Mocha theme and
  streaming (SSE) results.

## Quick start

### Docker (recommended)

```sh
docker build -t gosearx .
docker run --rm -p 8080:8080 \
  -e GITHUB_TOKEN=...  -e BRAVE_API_KEY=... \
  -v ./settings.yml:/app/settings.yml:ro \
  gosearx
# open http://localhost:8080
```

Prebuilt images are published to GHCR on every push to `main`:

```sh
docker run --rm -p 8080:8080 ghcr.io/abs3ntdev/gosearx:latest
```

Valkey/Redis runs **externally** — set `VALKEY_URL` (e.g.
`valkey://valkey:6379/1`), or leave it unset to use the in-memory cache. See
`docker-compose.yml` for an example.

**Unraid:** see [`deploy/unraid/`](deploy/unraid/) for a Community-Apps template
and a step-by-step guide.

### From source

```sh
make all          # builds the frontend then the Go binary
./gosearx serve   # reads settings.yml; serves UI + API

# one-shot CLI search (parallel, merged, scored)
./gosearx search -q "go programming language"
./gosearx search -q "!gh ripgrep"      # !bang selects an engine/shortcut

# frontend dev server (proxies the API)
cd web && npm run dev
```

Requires Go 1.26+ and Node 22+. `make all` runs `npm run build` (embedding the
SPA) then `go build`.

## Configuration

`settings.yml` configures the instance; engine *definitions* live in `engines/`
and are overridden here by name. **Secrets are read from the environment** via
`${VAR}` expansion — never commit credentials:

```yaml
general: { instance_name: "gosearx" }
server: { address: ":8080", engines_dir: engines }
search:
  default_category: general
  collapse_duplicates: true
valkey:
  url: "valkey://localhost:6379/1"   # empty => in-memory cache
ai:
  enabled: false
  provider: ollama                   # or openai-compatible
  model: "llama3.2"
engines:
  - name: braveapi
    config: { api_key: "${BRAVE_API_KEY}" }
  - name: github
    config: { token: "${GITHUB_TOKEN}" }
```

## Writing engines & plugins

Adding an engine or plugin **never requires recompiling** (except native Go).
Drop a file in `engines/` or `plugins/` and it loads on start.

A Lua engine returns the request (the host performs all I/O) and parses the
response:

```lua
function request(query, params)
  params.url = "https://example.com/search?" .. url.encode({ q = query })
  return params
end

function response(resp)
  local results, dom = {}, html.parse(resp.text)
  for _, r in ipairs(xpath.list(dom, '//li/a')) do
    results[#results+1] = { url = xpath.text(r, './@href'), title = xpath.text(r, '.') }
  end
  return results
end
```

Sandboxed API: `html.parse`, `xpath.{list,first,text,attr,url}`,
`url.{encode,escape,unescape}`, `json.{decode,encode}`, `base64.{encode,decode}`,
plus a controlled `http.get`/`http.post` for engines. Denied: filesystem,
process, and arbitrary network access.

See **[`docs/engines.md`](docs/engines.md)** and
**[`docs/plugins.md`](docs/plugins.md)** for the full authoring guides (all
backends), and `engines/hackernews.mjs`, `plugins/morse.mjs`,
`plugins/weather.sh` for JS/exec examples.

### Custom plugins in Docker

Built-in plugins ship at `/app/plugins`. Mount your own at `/app/custom-plugins`
(they *add to*, never hide, the built-ins):

```sh
docker run -p 8080:8080 -v ./my-plugins:/app/custom-plugins:ro gosearx
```

Exec plugins (`.sh`/`.py`/…) need an interpreter — the default image includes
`bash`/`python3`/`curl`.

## Layout

```
cmd/gosearx/         CLI: `serve` (API + UI) and `search` (one-shot)
internal/
  search/            orchestrator: parallel engines + merge/dedup/score + plugins
  result/            typed result union (main/image/answer/quote/chart/github/…)
  engine/            engine contract + tiers:
    declarative/       yaml xpath + json engines (zero code)
    script/            Lua engine tier
    jsscript/          JavaScript (goja) engine tier
    execengine/        exec/subprocess engine tier
    loader/            dispatches by file extension
  plugin/            plugin tiers (native/Lua/JS/exec) + lifecycle hooks
  jsruntime/         shared sandboxed goja runtime + pool
  lua/               shared sandboxed Lua env + pool + html/xpath bindings
  ai/                LLM answer synthesis (ollama / openai-compatible)
  github/            GitHub intent routing + unified ranking
  finance/           pluggable market-data sources + ticker detection
  cache/             Valkey/Redis cache with in-memory fallback
  network/ query/ traits/ preferences/ limiter/ metrics/ proxy/ bangs/ config/
engines/             ~70 engine definitions (.yaml/.lua/.mjs/.sh)
plugins/             plugins (.lua/.mjs/.sh) + native Go plugins in internal/plugin/native
web/                 React + TS SPA; embedded via embed.FS
  src/results/registry.tsx   maps result type -> component (extensibility point)
docs/                engines.md, plugins.md (authoring guides)
```

## License

Derives from SearXNG (AGPL-3.0-or-later); this project is AGPL-3.0-or-later.
