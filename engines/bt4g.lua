-- BT4G — torrent search via its RSS feed (files category).
-- @shortcut: bt4g
-- @categories: files
-- @timeout: 6s

function request(query, params)
  params.url = "https://bt4gprx.com/search?" .. url.encode({
    q = query,
    orderby = "size",
    p = tostring(params.pageno or 1),
    page = "rss",
  })
  params.headers["Accept"] = "application/rss+xml, application/xml"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  for item in resp.text:gmatch("<item>(.-)</item>") do
    local title = item:match("<title>(.-)</title>")
    local link = item:match("<link>(.-)</link>")
    local size = item:match("<size>(.-)</size>") or item:match("<enclosure[^>]-length=\"(%d+)\"")
    local seeders = item:match("<seeders>(%d+)</seeders>")
    local leechers = item:match("<leechers>(%d+)</leechers>")
    if title and link then
      title = title:gsub("^<!%[CDATA%[(.*)%]%]>$", "%1")
      results[#results + 1] = {
        type = "torrent",
        title = title,
        url = link,
        fileSize = size or "",
        seeders = tonumber(seeders) or 0,
        leechers = tonumber(leechers) or 0,
      }
    end
  end
  return results
end
