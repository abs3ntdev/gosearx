-- Openverse (images) — Lua port of searx/engines/openverse.py.
-- Clean JSON API, no key. Emits type=image results for the images category.
-- @shortcut: opv
-- @categories: images
-- @timeout: 5s

local base = "https://api.openverse.org/v1/images/"

function request(query, params)
  -- Anonymous Openverse requests are capped at page_size=20.
  params.url = base .. "?" .. url.encode({
    q = query,
    page = tostring(params.pageno or 1),
    page_size = "20",
    format = "json",
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then
    error("openverse HTTP " .. tostring(resp.status_code))
  end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data.results or {}) do
    results[#results + 1] = {
      type = "image",
      title = r.title or "",
      url = r.foreign_landing_url or r.url or "",
      imgSrc = r.url or "",
      thumbnailSrc = r.thumbnail or r.url or "",
      source = r.provider or "",
    }
  end
  return results
end
