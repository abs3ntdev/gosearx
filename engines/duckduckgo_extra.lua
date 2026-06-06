-- DuckDuckGo images/videos/news — Lua port of duckduckgo_extra.py.
-- DDG requires a `vqd` token (scraped from the search page) before its
-- i.js/v.js/news.js JSON endpoints will answer. The host fetch loads the page
-- to grab vqd; response() then calls the JSON endpoint via http.get.
-- ddg_category (images|videos|news) comes from settings.yml config.
-- @shortcut: ddi
-- @categories: images
-- @timeout: 7s

local UA = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"

local path_map = { images = "i", videos = "v", news = "news" }

function request(query, params)
  -- load the HTML page that embeds vqd="..."
  params.url = "https://duckduckgo.com/?q=" .. url.escape(query) .. "&ia=web"
  params.headers["User-Agent"] = UA
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local cat = (resp.config and resp.config.ddg_category) or "images"
  local vqd = resp.text:match('vqd="([^"]+)"') or resp.text:match("vqd=([%d%-]+)")
  if not vqd then return results end

  local endpoint = "https://duckduckgo.com/" .. (path_map[cat] or "i") .. ".js?"
    .. url.encode({ l = "us-en", o = "json", q = resp.query or "", vqd = vqd, f = ",,,,,", p = "1" })
  local jr = http.get(endpoint, { ["User-Agent"] = UA, ["Accept"] = "application/json",
    ["Referer"] = "https://duckduckgo.com/" })
  if not jr or jr.status_code ~= 200 then return results end
  local data = json.decode(jr.text)
  if type(data) ~= "table" then return results end

  for _, r in ipairs(data.results or {}) do
    if cat == "images" then
      results[#results + 1] = {
        type = "image", title = r.title or "",
        url = r.url or r.image or "", imgSrc = r.image or "",
        thumbnailSrc = r.thumbnail or r.image or "", source = "DuckDuckGo",
      }
    elseif cat == "videos" then
      local thumb = ""
      if type(r.images) == "table" then thumb = r.images.small or r.images.medium or "" end
      results[#results + 1] = {
        type = "video", title = r.title or "",
        url = r.content or "", content = r.description or "", thumbnail = thumb,
      }
    elseif cat == "news" then
      results[#results + 1] = {
        title = r.title or "", url = r.url or "", content = r.excerpt or "",
      }
    end
  end
  return results
end
