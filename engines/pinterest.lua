-- Pinterest — images via the resource search API.
-- @shortcut: pin
-- @categories: images
-- @timeout: 6s

function request(query, params)
  local opts = json.encode({
    options = { query = query, bookmarks = {} },
  })
  params.url = "https://www.pinterest.com/resource/BaseSearchResource/get/?"
    .. url.encode({ source_url = "/search/pins/?q=" .. query, data = opts })
  params.headers["Accept"] = "application/json"
  params.headers["X-Requested-With"] = "XMLHttpRequest"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  local rr = (((data or {}).resource_response or {}).data or {}).results
  if type(rr) ~= "table" then return results end
  for _, p in ipairs(rr) do
    local img = ""
    if type(p.images) == "table" and type(p.images.orig) == "table" then img = p.images.orig.url end
    if img ~= "" then
      results[#results + 1] = {
        type = "image",
        title = p.grid_title or p.description or "Pin",
        url = "https://www.pinterest.com/pin/" .. tostring(p.id) .. "/",
        imgSrc = img,
        thumbnailSrc = img,
        source = "Pinterest",
      }
    end
  end
  return results
end
