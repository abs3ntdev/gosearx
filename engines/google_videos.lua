-- Google Videos — scrapes the video results page (tbm=vid) using the GSA UA.
-- @shortcut: gov
-- @categories: videos
-- @timeout: 6s

local UA =
  "Mozilla/5.0 (Linux; Android 5.0; SM-G900P Build/LRX21T) "
  .. "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.1255.1902 Mobile Safari/537.36 NSTNWV"

local function unwrap(href)
  if not href then return "" end
  local q = href:match("^/url%?q=([^&]+)")
  if q then return url.unescape(q) end
  if href:sub(1, 4) == "http" then return href end
  return ""
end

function request(query, params)
  local start = (params.pageno - 1) * 10
  params.url = "https://www.google.com/search?" .. url.encode({
    q = query, tbm = "vid", hl = "en", start = tostring(start),
  })
  params.headers["Accept"] = "*/*"
  params.headers["User-Agent"] = UA
  params.cookies["CONSENT"] = "YES+"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  if resp.url:find("/sorry") then return results end
  local dom = html.parse(resp.text)
  for _, a in ipairs(xpath.list(dom, '//a[@data-ved and not(@class)]')) do
    local target = unwrap(xpath.attr(a, ".", "href"))
    local title = xpath.text(a, ".//h3")
    if target ~= "" and target:sub(1, 4) == "http"
       and not target:find("google%.com/") and title ~= "" then
      results[#results + 1] = { type = "video", title = title, url = target }
    end
  end
  return results
end
