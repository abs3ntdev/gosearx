-- Chefkoch — recipe search via its public API (no key).
-- @shortcut: chef
-- @categories: general
-- @timeout: 5s

function request(query, params)
  local offset = (params.pageno - 1) * 20
  params.url = "https://api.chefkoch.de/v2/recipes?" .. url.encode({
    query = query, limit = "20", offset = tostring(offset),
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data.results or {}) do
    local rec = r.recipe or r
    if rec and rec.id then
      local img = ""
      if rec.previewImageUrlTemplate then
        img = rec.previewImageUrlTemplate:gsub("<format>", "crop-240x300")
      end
      results[#results + 1] = {
        title = rec.title or "",
        url = "https://www.chefkoch.de/rezepte/" .. rec.id .. "/",
        content = rec.subtitle or "",
        thumbnail = img,
      }
    end
  end
  return results
end
