-- Photon (Komoot) — map/place search (no key).
-- @shortcut: ph
-- @categories: map
-- @timeout: 5s

function request(query, params)
  params.url = "https://photon.komoot.io/api/?" .. url.encode({ q = query, limit = "10" })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, f in ipairs(data.features or {}) do
    local p = f.properties or {}
    if p.osm_id and p.osm_type then
      local parts = {}
      for _, k in ipairs({ "name", "street", "city", "state", "country" }) do
        if p[k] then parts[#parts + 1] = p[k] end
      end
      local title = table.concat(parts, ", ")
      local lat, lon = 0, 0
      if type(f.geometry) == "table" and type(f.geometry.coordinates) == "table" then
        lon = f.geometry.coordinates[1] or 0
        lat = f.geometry.coordinates[2] or 0
      end
      results[#results + 1] = {
        type = "map",
        title = title ~= "" and title or (p.name or "place"),
        url = "https://openstreetmap.org/" .. p.osm_type .. "/" .. tostring(math.floor(p.osm_id)),
        content = p.osm_value or "",
        address = title,
        latitude = lat,
        longitude = lon,
      }
    end
  end
  return results
end
