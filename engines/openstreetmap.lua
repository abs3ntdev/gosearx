-- OpenStreetMap (Nominatim) — map search (no key).
-- @shortcut: osm
-- @categories: map
-- @timeout: 6s

function request(query, params)
  params.url = "https://nominatim.openstreetmap.org/search?"
    .. url.encode({ q = query, format = "jsonv2", addressdetails = "1", limit = "10" })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data) do
    local osm_type = r.osm_type or "node"
    local osm_id = r.osm_id and tostring(math.floor(r.osm_id)) or ""
    results[#results + 1] = {
      type = "map",
      title = r.display_name or r.name or "",
      url = "https://www.openstreetmap.org/" .. osm_type .. "/" .. osm_id,
      content = r.type or "",
      address = r.display_name or "",
      latitude = tonumber(r.lat) or 0,
      longitude = tonumber(r.lon) or 0,
    }
  end
  return results
end
