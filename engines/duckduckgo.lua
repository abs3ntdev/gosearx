-- DuckDuckGo (HTML, no-JS endpoint) — Lua port of the HTML scraper path.
-- Uses the html.duckduckgo.com POST form which returns plain result HTML.
-- @shortcut: ddg
-- @categories: general, web
-- @timeout: 5s

function request(query, params)
  -- The lite/html endpoints accept POST form data; the host sends params.data.
  params.method = "POST"
  params.url = "https://html.duckduckgo.com/html/"
  params.data = url.encode({
    q = query,
    kl = "wt-wt", -- no region
  })
  params.headers["Content-Type"] = "application/x-www-form-urlencoded"
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  local dom = html.parse(resp.text)
  for _, r in ipairs(xpath.list(dom, '//div[contains(@class,"result__body")]')) do
    local link = xpath.text(r, './/a[contains(@class,"result__a")]/@href')
    local title = xpath.text(r, './/a[contains(@class,"result__a")]')
    local content = xpath.text(r, './/a[contains(@class,"result__snippet")]')
    if link ~= "" and title ~= "" then
      results[#results + 1] = { url = link, title = title, content = content }
    end
  end
  return results
end
