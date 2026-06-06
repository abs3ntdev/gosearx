# Writing plugins

A *plugin* hooks into the search lifecycle. Plugins live in the `plugins/`
directory and load automatically — no recompile needed. They can be written in
several languages (see [Backends & security](#backends--security)); the examples
below use Lua, but the hook contract is the same for all.

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

## Backends & security

Plugins are dispatched by file extension:

| Backend | Ext | Where it runs | Isolation |
|---|---|---|---|
| Lua | `.lua` | in-process (gopher-lua) | **Sandboxed** — no fs/process/arbitrary-net |
| JavaScript | `.mjs` | in-process (goja) | **Sandboxed** — no fs/process/net |
| Native Go | — | compiled in | Full trust (ships in the binary) |
| Exec script | `.sh` `.py` `.rb` `.pl` `.plugin` | subprocess | **Not sandboxed** (see below) |

JS plugins use camelCase hooks (`preSearch`/`onResult`/`postSearch`) and a
`keywords` array; otherwise they mirror Lua. Exec plugins speak a JSON protocol
on stdin/stdout (one object per hook); see `plugins/weather.sh` and the
`internal/plugin/exec.go` header for the schema.

### Exec plugin security model

Exec plugins run an **arbitrary external program**, so they are powerful and, by
design, **not sandboxed**: the script executes with the gosearx process's own
privileges (it can read files the process can read, open sockets, spawn
processes, etc.). Treat exec plugins like any other operator-supplied code —
**only install scripts you trust**, the same way you'd trust the binary or
`settings.yml`.

What the host *does* guarantee, so that a remote searcher can never abuse them:

- **No command/shell injection from queries.** The search query, client IP, and
  user-agent reach the script **only as JSON on stdin** — never as command-line
  arguments and never via `sh -c`. A query like `; rm -rf /` is just a string the
  script reads. *(Verified by `TestExec_NoCommandInjection`.)*
- **Scrubbed environment.** The child sees only `PATH` and a marker var — your
  `GITHUB_TOKEN`, `BRAVE_API_KEY`, etc. are **not** inherited.
  *(`TestExec_EnvScrubbed`.)*
- **Bounded output.** stdout is capped (4 MiB); a flood is rejected, not OOM'd.
  *(`TestExec_OutputCap`.)*
- **Timeout + process-group kill.** Each call has a deadline (default 5s,
  `@timeout:` overridable); on expiry the whole child process group — including
  any grandchildren like a hung `sleep` — is SIGKILLed. *(`TestExec_Timeout`.)*

The trust boundary, in short: **a searcher cannot make an exec plugin do
anything; an operator who installs a malicious plugin can.** If you need to run
untrusted plugins, prefer the in-process Lua/JS backends (sandboxed), or run the
container with additional OS-level confinement (read-only rootfs, dropped
capabilities, seccomp, a non-root user — the image already runs as uid 65532).
