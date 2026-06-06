-- Bing News — Lua port of bing_news.py via the infinite-scroll AJAX endpoint.
-- @shortcut: bin
-- @categories: news
-- @timeout: 5s

function request(query, params)
  local first = (params.pageno - 1) * 10 + 1
  params.url = "https://www.bing.com/news/infinitescrollajax?" .. url.encode({
    q = query, first = tostring(first), InfiniteScroll = "1",
  })
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local dom = html.parse(resp.text)
  for _, a in ipairs(xpath.list(dom, '//div[contains(@class,"news-card")]//a[@class="title"] | //div[@class="newsitem"]//a')) do
    local href = xpath.attr(a, ".", "href")
    local title = tostring(a)
    if href ~= "" and href:sub(1, 4) == "http" and title ~= "" then
      results[#results + 1] = { title = title, url = href, content = "" }
    end
  end
  return results
end
