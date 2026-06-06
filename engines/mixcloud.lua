-- Mixcloud — music, official public API (no key).
-- @shortcut: mc
-- @categories: music
-- @timeout: 5s

function request(query, params)
  params.url = "https://api.mixcloud.com/search/?"
    .. url.encode({
      q = query,
      type = "cloudcast",
      limit = "20",
      offset = tostring((params.pageno - 1) * 20),
    })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data.data or {}) do
    results[#results + 1] = {
      title = r.name or "",
      url = r.url or "",
      content = (r.user and r.user.name) or "",
    }
  end
  return results
end
