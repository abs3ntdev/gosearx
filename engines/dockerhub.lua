-- Docker Hub — Lua port of searx/engines/docker_hub.py (JSON API).
-- @shortcut: dh
-- @categories: it, packages
-- @timeout: 5s

function request(query, params)
  params.url = "https://hub.docker.com/api/search/v3/catalog/search?"
    .. url.encode({ query = query, from = tostring((params.pageno - 1) * 25), size = "25" })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then error("dockerhub HTTP " .. tostring(resp.status_code)) end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data.results or {}) do
    local name = r.id or r.name or ""
    local slug = name
    if r.type == "image" and not name:find("/") then slug = "_/" .. name end
    results[#results + 1] = {
      title = name,
      url = "https://hub.docker.com/r/" .. (name:find("/") and name or ("_/" .. name)),
      content = (r.short_description or r.description or ""),
    }
  end
  return results
end
