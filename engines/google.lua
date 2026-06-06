-- Google (web) — Lua port of searx/engines/google.py.
--
-- Google serves a JS-required wall to modern desktop user-agents. The trick
-- (same as SearXNG) is to send a Google-Search-App style Android user-agent +
-- a CONSENT cookie, which returns the legacy no-JS mobile results layout where
-- each result is an <a href="/url?q=REAL_URL..." data-ved ...> with no class.
-- @shortcut: go
-- @categories: general, web
-- @timeout: 5s

-- An older Android UA reliably returns the scrapeable layout. Override via
-- settings.yml config.user_agent if Google rotates this.
local DEFAULT_UA =
  "Mozilla/5.0 (Linux; Android 5.0; SM-G900P Build/LRX21T) "
  .. "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.1255.1902 Mobile Safari/537.36 NSTNWV"

local time_range_map = { day = "d", week = "w", month = "m", year = "y" }
local safe_map = { [0] = "off", [1] = "medium", [2] = "high" }

function request(query, params)
  local start = (params.pageno - 1) * 10
  local q = "https://www.google.com/search?" .. url.encode({
    q = query,
    hl = "en",
    ie = "utf8",
    oe = "utf8",
    filter = "0",
    start = tostring(start),
    num = "20",
  })
  if params.time_range and time_range_map[params.time_range] then
    q = q .. "&tbs=qdr:" .. time_range_map[params.time_range]
  end
  if params.safesearch and safe_map[params.safesearch] then
    q = q .. "&safe=" .. safe_map[params.safesearch]
  end
  params.url = q
  params.headers["Accept"] = "*/*"
  params.headers["User-Agent"] = (params.config and params.config.user_agent) or DEFAULT_UA
  params.cookies["CONSENT"] = "YES+"
  return params
end

-- unwrap "/url?q=REAL&sa=..." -> REAL (Google's redirect wrapper).
local function unwrap(href)
  if not href then return "" end
  local q = href:match("^/url%?q=([^&]+)")
  if q then
    return url.unescape(q)
  end
  if href:sub(1, 4) == "http" then return href end
  return ""
end

function response(resp)
  local results = {}
  -- bot-protection page
  if resp.url:find("/sorry") or (#resp.text < 2000 and resp.text:find("/sorry/")) then
    return results
  end

  local dom = html.parse(resp.text)
  -- Legacy layout: result anchors carry data-ved and no class attribute.
  for _, a in ipairs(xpath.list(dom, '//a[@data-ved and not(@class)]')) do
    local href = xpath.attr(a, ".", "href")
    local target = unwrap(href)
    if target ~= "" and target:sub(1, 4) == "http"
       and not target:find("google%.com/") then
      -- Title lives in the h3 inside the anchor; the breadcrumb is a sibling
      -- div, so extracting h3 text avoids the "...www.site.com ›" cruft.
      local title = xpath.text(a, ".//h3")
      if title == "" then title = tostring(a) end
      if title ~= "" then
        results[#results + 1] = { url = target, title = title }
      end
    end
  end
  return results
end
