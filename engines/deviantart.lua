-- DeviantArt — images. Scrapes the results page for deviation thumbnails.
-- DeviantArt uses obfuscated class names, so we target anchors linking to
-- /art/ pages that contain a thumbnail image.
-- @shortcut: da
-- @categories: images
-- @timeout: 6s

function request(query, params)
  params.url = "https://www.deviantart.com/search?" .. url.encode({ q = query })
  params.headers["Accept"] = "text/html"
  params.headers["User-Agent"] = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local dom = html.parse(resp.text)
  local seen = {}
  for _, a in ipairs(xpath.list(dom, '//a[contains(@href,"/art/")]')) do
    local href = xpath.attr(a, ".", "href")
    local img = xpath.attr(a, ".//img", "src")
    local title = xpath.attr(a, ".//img", "alt")
    if href ~= "" and img ~= "" and not seen[href] then
      seen[href] = true
      results[#results + 1] = {
        type = "image",
        title = title ~= "" and title or "DeviantArt",
        url = href,
        imgSrc = img,
        thumbnailSrc = img,
        source = "DeviantArt",
      }
    end
  end
  return results
end
