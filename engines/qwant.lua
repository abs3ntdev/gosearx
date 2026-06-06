-- Qwant — Lua port of qwant.py (JSON API v3). qwant_categ (web|images|videos|
-- news) comes from config. Returns typed results per category.
-- @shortcut: qw
-- @categories: general, web
-- @timeout: 6s

local UA = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"

function request(query, params)
  local categ = (params.config and params.config.qwant_categ) or "web"
  local count = 10
  local offset = (params.pageno - 1) * count
  params.url = "https://api.qwant.com/v3/search/" .. categ .. "?" .. url.encode({
    q = query, count = tostring(count), offset = tostring(offset),
    locale = "en_US", safesearch = tostring(params.safesearch or 0), t = categ,
  })
  params.headers["User-Agent"] = UA
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local categ = (resp.config and resp.config.qwant_categ) or "web"
  local data = json.decode(resp.text)
  local mainline = (((data or {}).data or {}).result or {}).items
  if type(mainline) ~= "table" then return results end
  -- items may be {mainline={...}} or a flat array
  local items = mainline.mainline or mainline
  if type(items) ~= "table" then return results end

  local function walk(list)
    for _, it in ipairs(list) do
      if it.type == "web" or it.url then
        if categ == "images" then
          results[#results + 1] = { type = "image", title = it.title or "",
            url = it.url or "", imgSrc = it.media or it.media_fullsize or "",
            thumbnailSrc = it.thumbnail or it.media or "", source = "Qwant" }
        elseif categ == "videos" then
          results[#results + 1] = { type = "video", title = it.title or "",
            url = it.url or "", thumbnail = it.thumbnail or "" }
        else
          results[#results + 1] = { title = it.title or "", url = it.url or "",
            content = it.desc or it.description or "" }
        end
      elseif it.items then
        walk(it.items)
      end
    end
  end
  walk(items)
  return results
end
