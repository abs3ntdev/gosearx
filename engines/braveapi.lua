-- Brave Search API (web) — Lua port of searx/engines/braveapi.py
-- Requires an API key in settings.yml:
--   engines:
--     - name: braveapi
--       config: { api_key: "YOUR-KEY" }
-- @shortcut: br
-- @categories: general, web
-- @timeout: 5s

local base_url = "https://api.search.brave.com/res/v1/web/search"
local results_per_page = 20

local time_range_map = {
  day = "past_day", week = "past_week", month = "past_month", year = "past_year",
}

function request(query, params)
  local args = {
    q = query,
    count = tostring(results_per_page),
    offset = tostring((params.pageno - 1) * results_per_page),
  }
  if params.time_range and time_range_map[params.time_range] then
    args.time_range = time_range_map[params.time_range]
  end
  if params.safesearch and params.safesearch > 0 then
    args.safesearch = "strict"
  end
  params.url = base_url .. "?" .. url.encode(args)
  params.headers["Accept"] = "application/json"
  params.headers["X-Subscription-Token"] = params.config.api_key or ""
  return params
end

function response(resp)
  local results = {}
  local data = json.decode(resp.text)
  if type(data) ~= "table" or type(data.web) ~= "table" then
    return results
  end
  for _, r in ipairs(data.web.results or {}) do
    local thumb = ""
    if type(r.thumbnail) == "table" then thumb = r.thumbnail.src or "" end
    results[#results + 1] = {
      url = r.url,
      title = r.title,
      content = r.description or "",
      thumbnail = thumb,
      publishedDate = r.age or "",
    }
  end
  return results
end
