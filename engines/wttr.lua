-- wttr.in — weather. Only triggers on an explicit "weather <place>" query (or
-- the !wttr bang), so it never fires on unrelated searches.
-- @shortcut: wttr
-- @categories: general, weather
-- @timeout: 8s

-- extractPlace returns the place if the query is a weather request, else nil.
local function extractPlace(q)
  q = (q or ""):gsub("^%s+", ""):gsub("%s+$", "")
  local p = q:match("^[Ww]eather%s+in%s+(.+)$")
    or q:match("^[Ww]eather%s+(.+)$")
    or q:match("^(.+)%s+weather$")
  return p
end

function request(query, params)
  local place = extractPlace(query)
  if not place then
    -- Not a weather query: hit a trivial URL and emit nothing in response.
    params.url = "https://wttr.in/?format=j1"
    params.config = params.config or {}
    params.config._skip = "1"
    return params
  end
  params.url = "https://wttr.in/" .. url.escape(place) .. "?format=j1&lang=en"
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  -- Re-check intent from the echoed query so a non-weather search shows nothing.
  if not extractPlace(resp.query) then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  local cur = (data.current_condition or {})[1]
  local area = ((data.nearest_area or {})[1] or {})
  if cur then
    local place = ""
    if area.areaName and area.areaName[1] then place = area.areaName[1].value end
    local desc = ""
    if cur.weatherDesc and cur.weatherDesc[1] then desc = cur.weatherDesc[1].value end
    local ans = string.format("Weather in %s: %s, %s°C (feels %s°C), humidity %s%%, wind %s km/h",
      place, desc, cur.temp_C or "?", cur.FeelsLikeC or "?", cur.humidity or "?", cur.windspeedKmph or "?")
    results[#results + 1] = { type = "answer", answer = ans }
  end
  return results
end
