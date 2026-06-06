-- Google Images — Lua port of searx/engines/google_images.py. Uses the
-- tbm=isch + async=_fmt:json trick that returns an embedded JSON blob.
-- @shortcut: goi
-- @categories: images
-- @timeout: 6s

local DEFAULT_UA =
  "Mozilla/5.0 (Linux; Android 5.0; SM-G900P Build/LRX21T) "
  .. "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.1255.1902 Mobile Safari/537.36 NSTNWV"

function request(query, params)
  params.url = "https://www.google.com/search?"
    .. url.encode({ q = query, tbm = "isch", hl = "en", asearch = "isch" })
    .. "&async=_fmt:json,p:1,ijn:" .. tostring((params.pageno or 1) - 1)
  params.headers["Accept"] = "*/*"
  params.headers["User-Agent"] = (params.config and params.config.user_agent) or DEFAULT_UA
  params.cookies["CONSENT"] = "YES+"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local start = resp.text:find('{"ischj":', 1, true)
  if not start then return results end
  local data = json.decode(resp.text:sub(start))
  if type(data) ~= "table" or type(data.ischj) ~= "table" then return results end
  for _, item in ipairs(data.ischj.metadata or {}) do
    local orig = item.original_image or {}
    local thumb = item.thumbnail or {}
    local res = item.result or {}
    if orig.url then
      results[#results + 1] = {
        type = "image",
        title = res.page_title or res.site_title or "image",
        url = res.referrer_url or orig.url,
        imgSrc = orig.url,
        thumbnailSrc = thumb.url or orig.url,
        source = res.site_title or "Google Images",
      }
    end
  end
  return results
end
