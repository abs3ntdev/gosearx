-- Google Scholar — scientific papers. Scrapes the scholar results page.
-- @shortcut: gos
-- @categories: science
-- @timeout: 6s

local UA =
  "Mozilla/5.0 (Linux; Android 5.0; SM-G900P Build/LRX21T) "
  .. "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.1255.1902 Mobile Safari/537.36 NSTNWV"

function request(query, params)
  local start = (params.pageno - 1) * 10
  params.url = "https://scholar.google.com/scholar?" .. url.encode({
    q = query, start = tostring(start), hl = "en",
  })
  params.headers["Accept"] = "*/*"
  params.headers["User-Agent"] = UA
  params.cookies["CONSENT"] = "YES+"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  if resp.text:find("gs_captcha_f") then return results end
  local dom = html.parse(resp.text)
  for _, r in ipairs(xpath.list(dom, '//div[@class="gs_r gs_or gs_scl"] | //div[contains(@class,"gs_ri")]')) do
    local link = xpath.attr(r, './/h3[@class="gs_rt"]/a', "href")
    local title = xpath.text(r, './/h3[@class="gs_rt"]')
    local content = xpath.text(r, './/div[@class="gs_rs"]')
    local meta = xpath.text(r, './/div[@class="gs_a"]')
    if link ~= "" and title ~= "" then
      results[#results + 1] = { url = link, title = title,
        content = (meta ~= "" and (meta .. " — ") or "") .. content }
    end
  end
  return results
end
