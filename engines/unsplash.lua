-- Unsplash — images via the public napi search endpoint (no key).
-- @shortcut: us
-- @categories: images
-- @timeout: 5s

function request(query, params)
  params.url = "https://unsplash.com/napi/search/photos?"
    .. url.encode({ query = query, page = tostring(params.pageno or 1), per_page = "20" })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data.results or {}) do
    local urls = r.urls or {}
    results[#results + 1] = {
      type = "image",
      title = r.description or r.alt_description or "Unsplash photo",
      url = (r.links and r.links.html) or "",
      imgSrc = urls.regular or urls.full or "",
      thumbnailSrc = urls.thumb or urls.small or "",
      source = "Unsplash",
    }
  end
  return results
end
