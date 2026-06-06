-- StackExchange (Stack Overflow by default) — Lua port of stackexchange.py.
-- The api_site is configurable via settings.yml config.api_site so the same
-- engine file serves stackoverflow / askubuntu / superuser.
-- @shortcut: st
-- @categories: it, q&a
-- @timeout: 5s

function request(query, params)
  local site = params.config.api_site or "stackoverflow"
  params.url = "https://api.stackexchange.com/2.3/search/advanced?"
    .. url.encode({
      order = "desc",
      sort = "relevance",
      q = query,
      site = site,
      page = tostring(params.pageno or 1),
    })
  params.headers["Accept"] = "application/json"
  return params
end

-- Minimal HTML entity unescape for titles.
local function unescape(s)
  s = s:gsub("&amp;", "&"):gsub("&lt;", "<"):gsub("&gt;", ">")
  s = s:gsub("&quot;", '"'):gsub("&#39;", "'"):gsub("&#x27;", "'")
  return s
end

function response(resp)
  local results = {}
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, item in ipairs(data.items or {}) do
    results[#results + 1] = {
      url = item.link,
      title = unescape(item.title or ""),
      content = (item.is_answered and "answered" or "unanswered")
        .. " · score " .. tostring(item.score or 0),
    }
  end
  return results
end
