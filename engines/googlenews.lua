-- Google News — uses the public Google News RSS feed (no token, reliable).
-- Emits standard results in the news category.
-- @shortcut: gon
-- @categories: news
-- @timeout: 5s

function request(query, params)
  params.url = "https://news.google.com/rss/search?" .. url.encode({
    q = query,
    hl = "en-US",
    gl = "US",
    ceid = "US:en",
  })
  params.headers["Accept"] = "application/rss+xml, application/xml, text/xml"
  return params
end

-- crude XML field extractor (the feed is simple RSS).
local function strip_cdata(s)
  return (s:gsub("^<!%[CDATA%[(.*)%]%]>$", "%1"))
end

function response(resp)
  local results = {}
  local text = resp.text
  -- iterate <item>...</item> blocks
  for item in text:gmatch("<item>(.-)</item>") do
    local title = item:match("<title>(.-)</title>")
    local link = item:match("<link>(.-)</link>")
    local pub = item:match("<pubDate>(.-)</pubDate>")
    if title and link then
      results[#results + 1] = {
        title = strip_cdata(title),
        url = link,
        content = pub or "",
        publishedDate = pub or "",
      }
    end
  end
  return results
end
