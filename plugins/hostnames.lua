-- hostnames: rewrite or remove result hostnames. Port of SearXNG's hostnames
-- plugin (on_result hook). Configure the rules below: `replace` maps a Lua
-- pattern on the host to a replacement host; `remove` drops matching results.
-- @about: Rewrite result hostnames (e.g. youtube.com -> an invidious instance).

-- Example rules (edit to taste). Patterns are Lua patterns matched on the host.
local replace = {
  -- ["youtube%.com$"] = "yt.example.com",
  -- ["reddit%.com$"] = "teddit.example.com",
}
local remove = {
  -- "facebook%.com$",
}

local function host_of(u)
  return u:match("^%w+://([^/]+)")
end

function on_result(result)
  local url = result.url or ""
  local host = host_of(url)
  if not host then return true end

  for _, pat in ipairs(remove) do
    if host:match(pat) then
      return false -- drop result
    end
  end
  for pat, repl in pairs(replace) do
    if host:match(pat) then
      result.url = url:gsub(host, repl, 1)
      break
    end
  end
  return true
end
