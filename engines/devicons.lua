-- Devicons — developer/tech icons, matches names against the query.
-- @shortcut: di
-- @categories: images
-- @timeout: 5s

local cdn = "https://cdn.jsdelivr.net/gh/devicons/devicon@latest"

function request(query, params)
  params.url = cdn .. "/devicon.json"
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  local q = (resp.query or ""):lower()
  for _, icon in ipairs(data) do
    local name = icon.name or ""
    local alts = type(icon.altnames) == "table" and table.concat(icon.altnames, " ") or ""
    if q == "" or name:lower():find(q, 1, true) or alts:lower():find(q, 1, true) then
      local versions = icon.versions or {}
      local svg = (versions.svg or {})[1] or "original"
      results[#results + 1] = {
        type = "image",
        title = name,
        url = "https://devicon.dev/",
        imgSrc = cdn .. "/icons/" .. name .. "/" .. name .. "-" .. svg .. ".svg",
        thumbnailSrc = cdn .. "/icons/" .. name .. "/" .. name .. "-" .. svg .. ".svg",
        source = "Devicons",
      }
      if #results >= 50 then break end
    end
  end
  return results
end
