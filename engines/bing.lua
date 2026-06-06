-- Bing (web) — Lua port of searx/engines/bing.py. Scrapes the HTML results
-- and unwraps Bing's ck/a redirect links (u=a1<base64url>).
-- @shortcut: bi
-- @categories: general, web
-- @timeout: 5s

function request(query, params)
  local first = (params.pageno - 1) * 10 + 1
  params.url = "https://www.bing.com/search?" .. url.encode({
    q = query, first = tostring(first),
  })
  params.headers["Accept"] = "text/html"
  params.headers["Accept-Language"] = "en-US,en;q=0.9"
  return params
end

-- base64url-decode (Bing uses URL-safe base64 without padding).
local function b64url_decode(s)
  s = s:gsub("-", "+"):gsub("_", "/")
  local pad = (4 - (#s % 4)) % 4
  s = s .. string.rep("=", pad)
  local ok, dec = pcall(base64.decode, s)
  if ok then return dec end
  return nil
end

local function unwrap(href)
  -- https://www.bing.com/ck/a?...&u=a1<base64url>&...
  local u = href:match("[?&]u=a1([^&]+)")
  if u then
    local dec = b64url_decode(url.unescape(u))
    if dec and dec:sub(1, 4) == "http" then return dec end
  end
  return href
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local dom = html.parse(resp.text)
  for _, item in ipairs(xpath.list(dom, '//ol[@id="b_results"]/li[contains(@class,"b_algo")]')) do
    local href = xpath.attr(item, ".//h2/a", "href")
    local title = xpath.text(item, ".//h2/a")
    local content = xpath.text(item, ".//p")
    if href ~= "" and title ~= "" then
      results[#results + 1] = { url = unwrap(href), title = title, content = content }
    end
  end
  return results
end
