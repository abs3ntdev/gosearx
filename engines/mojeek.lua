-- Mojeek (general/web) engine — Lua port of searx/engines/mojeek.py
-- @shortcut: mj
-- @categories: general, web
-- @timeout: 3s
-- Demonstrates the engine contract: request(query, params) builds the HTTP
-- request; response(resp) parses HTML via the xpath.* API into result rows.

local base_url = "https://www.mojeek.com"

-- XPath selectors mirror the Python engine exactly.
local results_xpath    = '//ul[@class="results-standard"]/li/a[@class="ob"]'
local url_xpath        = "./@href"
local title_xpath      = "../h2/a"
local content_xpath    = '..//p[@class="s"]'
local suggestion_xpath = '//div[@class="top-info"]/p[@class="top-info spell"]/em/a'

function request(query, params)
  local args = { q = query, safe = math.min(params.safesearch or 0, 1) }
  -- setting the page number on the first page triggers a rate-limit
  if (params.pageno or 1) > 1 then
    args.s = tostring(10 * (params.pageno - 1))
  end
  params.url = base_url .. "/search?" .. url.encode(args)
  return params
end

function response(resp)
  local results = {}
  local dom = html.parse(resp.text)

  for _, r in ipairs(xpath.list(dom, results_xpath)) do
    results[#results + 1] = {
      url     = xpath.text(r, url_xpath),
      title   = xpath.text(r, title_xpath),
      content = xpath.text(r, content_xpath),
    }
  end

  for _, s in ipairs(xpath.list(dom, suggestion_xpath)) do
    results[#results + 1] = { suggestion = tostring(s) }
  end

  return results
end
