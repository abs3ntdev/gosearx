-- Lucide icons — fetches the tag manifest and matches icon names/tags against
-- the query (images/icons). resp.query carries the search terms.
-- @shortcut: luc
-- @categories: images
-- @timeout: 5s

local cdn = "https://cdn.jsdelivr.net/npm/lucide-static"

function request(query, params)
  params.url = cdn .. "/tags.json"
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  local q = (resp.query or ""):lower()
  for name, tags in pairs(data) do
    local hay = name:lower()
    if type(tags) == "table" then hay = hay .. " " .. table.concat(tags, " "):lower() end
    if q == "" or hay:find(q, 1, true) then
      results[#results + 1] = {
        type = "image",
        title = name,
        url = "https://lucide.dev/icons/" .. name,
        imgSrc = cdn .. "/icons/" .. name .. ".svg",
        thumbnailSrc = cdn .. "/icons/" .. name .. ".svg",
        source = "Lucide",
      }
      if #results >= 60 then break end
    end
  end
  return results
end
