-- Wikimedia Commons — media search via the MediaWiki API. The media type
-- (image/video/audio/file) is set via config.media_type so one file backs the
-- wikicommons.images/videos/audio/files engines.
-- @shortcut: wc
-- @categories: images
-- @timeout: 6s

local api = "https://commons.wikimedia.org/w/api.php"

local filetype = {
  image = "bitmap|drawing",
  video = "video",
  audio = "audio",
  file = "office|archive",
}

function request(query, params)
  local mt = (params.config and params.config.media_type) or "image"
  params.url = api .. "?" .. url.encode({
    action = "query",
    format = "json",
    generator = "search",
    gsrnamespace = "6",
    gsrsearch = "filetype:" .. (filetype[mt] or "bitmap") .. " " .. query,
    gsrlimit = "20",
    prop = "imageinfo",
    iiprop = "url|size|mime",
    iiurlwidth = "320",
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  local pages = (((data or {}).query or {}).pages) or {}
  for _, p in pairs(pages) do
    local ii = (p.imageinfo or {})[1]
    if ii and ii.url then
      local mime = ii.mime or ""
      if mime:find("^image/") then
        results[#results + 1] = {
          type = "image", title = p.title or "",
          url = ii.descriptionurl or ii.url, imgSrc = ii.url,
          thumbnailSrc = ii.thumburl or ii.url, source = "Wikimedia Commons",
        }
      elseif mime:find("^video/") then
        results[#results + 1] = {
          type = "video", title = p.title or "",
          url = ii.descriptionurl or ii.url, thumbnail = ii.thumburl or "",
        }
      else
        results[#results + 1] = {
          title = p.title or "", url = ii.descriptionurl or ii.url, content = mime,
        }
      end
    end
  end
  return results
end
