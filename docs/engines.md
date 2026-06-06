# Writing engines

An *engine* integrates an upstream search provider. gosearx has three tiers;
pick the simplest that fits. Adding an engine **never requires recompiling** the
Go binary (except native engines).

Drop engine files in the `engines/` directory. They load automatically.

| Tier | When to use | File |
|---|---|---|
| Declarative XPath | Scrapes an HTML results page | `name.yaml` (`engine: xpath`) |
| Declarative JSON | Hits a JSON API | `name.yaml` (`engine: json`) |
| Lua script | Needs logic (tokens, multiple requests, custom parsing) | `name.lua` |
| Native Go | Performance-critical / very complex | `internal/engine/native/` |

## Tier 1 — Declarative XPath (zero code)

```yaml
# engines/example.yaml
name: example
engine: xpath
shortcut: ex
categories: [general]

search_url: https://example.com/search?q={query}&page={pageno}
paging: true
page_size: 10

results_xpath: '//div[@class="result"]'
url_xpath: './/a/@href'
title_xpath: './/a[@class="title"]'
content_xpath: './/p[@class="snippet"]'
suggestion_xpath: '//div[@class="suggestion"]'
```

URL placeholders: `{query}`, `{pageno}`, `{lang}`, `{time_range}`, `{safe_search}`.
Selectors use XPath (relative `.//` selectors resolve against each result node).

## Tier 2 — Declarative JSON

```yaml
# engines/mdn.yaml
name: mdn
engine: json
shortcut: mdn
search_url: https://developer.mozilla.org/api/v1/search?q={query}&page={pageno}
paging: true
results_query: documents       # slash-path to the results array
url_query: mdn_url
url_prefix: https://developer.mozilla.org
title_query: title
content_query: summary
suggestion_query: suggestions
```

`*_query` values are slash-separated paths into the decoded JSON
(`a/b/c`), with recursive descent (same semantics as SearXNG's json_engine).

## Tier 3 — Lua script

Define `request(query, params)` and `response(resp)`. The host performs the
HTTP request from what `request` returns; engines stay pure (no network/IO).

```lua
-- engines/example.lua
-- @shortcut: ex
-- @categories: general
-- @timeout: 3s

function request(query, params)
  params.url = "https://example.com/search?" .. url.encode({ q = query, p = params.pageno })
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  local dom = html.parse(resp.text)
  for _, r in ipairs(xpath.list(dom, '//div[@class="result"]')) do
    results[#results + 1] = {
      url = xpath.text(r, './/a/@href'),
      title = xpath.text(r, './/h3'),
      content = xpath.text(r, './/p'),
    }
  end
  return results
end
```

### Lua API available to engines

| Module | Functions |
|---|---|
| `html` | `html.parse(text) -> node` |
| `xpath` | `list(node, expr)`, `first(node, expr)`, `text(node, expr)`, `attr(node, expr, name)`, `url(node, expr, base)` |
| `url` | `url.encode(table|string)`, `url.escape(string)` |
| `json` | `json.decode(text)`, `json.encode(value)` |
| `base64` | `base64.encode/decode` |

Denied (sandbox): `os`, `io`, `require`, `load*`, filesystem, raw network.

### Result rows

Return an array of tables. A row is a main result by default:

```lua
{ url = "...", title = "...", content = "...", thumbnail = "..." }
```

Special rows by key/`type`:

```lua
{ suggestion = "..." }                       -- search suggestion
{ type = "answer", answer = "...", url = "" } -- instant answer
{ type = "image", imgSrc = "...", thumbnailSrc = "...", url = "...", title = "..." }
{ type = "quote", symbol = "AAPL", price = 190.2, currency = "USD", changePct = 1.3 }
{ type = "chart", symbol = "AAPL", chartKind = "candlestick",
  series = { { t = 1733000000, o = 1, h = 2, l = 0.5, c = 1.5 } } }
```

## Per-engine config overrides (`settings.yml`)

```yaml
engines:
  - name: example
    disabled: false
    weight: 1.5      # scoring weight
    timeout: 4s
    shortcut: ex
    categories: [general, web]
```
