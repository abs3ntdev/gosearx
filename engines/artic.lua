-- Art Institute of Chicago — public artwork image API (no key).
-- @shortcut: arc
-- @categories: images
-- @timeout: 5s

function request(query, params)
  params.url = "https://api.artic.edu/api/v1/artworks/search?" .. url.encode({
    q = query,
    query = "", -- placeholder; q drives it
    fields = "id,title,image_id,artist_display,date_display",
    limit = "20",
    page = tostring(params.pageno or 1),
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  local iiif = (data.config and data.config.iiif_url) or "https://www.artic.edu/iiif/2"
  for _, a in ipairs(data.data or {}) do
    if a.image_id then
      local img = iiif .. "/" .. a.image_id .. "/full/843,/0/default.jpg"
      results[#results + 1] = {
        type = "image",
        title = a.title or "",
        url = "https://www.artic.edu/artworks/" .. tostring(a.id),
        imgSrc = img,
        thumbnailSrc = iiif .. "/" .. a.image_id .. "/full/200,/0/default.jpg",
        content = (a.artist_display or "") .. " " .. (a.date_display or ""),
        source = "Art Institute of Chicago",
      }
    end
  end
  return results
end
