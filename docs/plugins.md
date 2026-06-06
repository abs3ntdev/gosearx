# Writing plugins

A *plugin* hooks into the search lifecycle. Plugins are Lua scripts in the
`plugins/` directory; they load automatically. No recompile needed.

## Hooks

Define any subset of these functions:

```lua
-- runs BEFORE engines; return false to abort the whole search
function pre_search(ctx)
  -- ctx.query : the search text
  return true
end

-- runs per main result; return false to DROP it; mutate the table to rewrite
function on_result(result)
  -- result.title / result.url / result.content / result.engine
  return true
end

-- runs AFTER engines; return an array of result rows to ADD (answers, etc.)
function post_search(ctx)
  return { { type = "answer", answer = "hello" } }
end
```

## Keyword-gated plugins

Declare a `keywords` table; the plugin's hooks only run when the first query
term matches one of them (case-insensitive). Useful for command-style features.

```lua
keywords = { "base64", "unbase64" }

function post_search(ctx)
  local cmd, rest = ctx.query:match("^(%S+)%s+(.+)$")
  if cmd == "base64" then
    return { { type = "answer", answer = base64.encode(rest) } }
  end
  return {}
end
```

## API available to plugins

Same sandboxed stdlib as engines: `url`, `json`, `base64`, plus `html`/`xpath`
(for plugins that post-process result pages). No `os`/`io`/network.

## Examples shipped

- `plugins/calculator.lua` — evaluates arithmetic (`2 + 3 * 4`) into an answer.
  A self-contained recursive-descent parser (the sandbox forbids `load`).
- `plugins/tracker_remover.lua` — strips `utm_*`, `fbclid`, … from result URLs
  (an `on_result` hook).
- `plugins/hash.lua` — keyword-gated base64 encode/decode.

## How hooks compose

- `pre_search`: all applicable plugins run; if any returns false, search aborts.
- `on_result`: every result passes through all (non-keyword) plugins; first
  `false` drops it. Mutations to `title/url/content` are written back.
- `post_search`: results from all applicable plugins are merged into the
  response (answers, quotes, charts, …).

Plugin errors are isolated: a failing plugin is skipped, never breaks search.
