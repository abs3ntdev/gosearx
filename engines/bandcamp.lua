-- Bandcamp — music search (scrapes the results page).
-- @shortcut: bc
-- @categories: music
-- @timeout: 6s

function request(query, params)
  params.url = "https://bandcamp.com/search?" .. url.encode({ q = query, page = tostring(params.pageno or 1) })
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local dom = html.parse(resp.text)
  for _, li in ipairs(xpath.list(dom, '//li[contains(@class,"searchresult")]')) do
    local link = xpath.text(li, './/div[contains(@class,"itemurl")]/a/@href')
    local title = xpath.text(li, './/div[contains(@class,"heading")]/a')
    local sub = xpath.text(li, './/div[contains(@class,"subhead")]')
    if link ~= "" and title ~= "" then
      results[#results + 1] = { title = title, url = link, content = sub }
    end
  end
  return results
end
