-- hash: keyword-triggered plugin that base64-encodes/decodes the rest of the
-- query. Demonstrates keyword gating and the post_search hook.
-- Triggers only when the first query term is "base64" or "unbase64".
-- @about: Encode/decode base64 from the search bar (e.g. "base64 hello").

keywords = { "base64", "unbase64" }

function post_search(ctx)
  local q = ctx.query
  local cmd, rest = q:match("^(%S+)%s+(.+)$")
  if not cmd then return {} end

  if cmd == "base64" then
    return { { type = "answer", answer = base64.encode(rest) } }
  elseif cmd == "unbase64" then
    local ok, decoded = pcall(base64.decode, rest)
    if ok then
      return { { type = "answer", answer = decoded } }
    end
  end
  return {}
end
