-- Vimeo — videos. Scrapes the search page's embedded JSON app state.
-- @shortcut: vm
-- @categories: videos
-- @timeout: 6s

function request(query, params)
  params.url = "https://vimeo.com/search/page:" .. tostring(params.pageno or 1)
    .. "?" .. url.encode({ q = query })
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  -- Vimeo embeds results as JSON in the page; extract /video/<id> links + titles.
  for id, title in resp.text:gmatch('"clip_id":(%d+).-"name":"(.-[^\\])"') do
    results[#results + 1] = {
      type = "video",
      title = (title:gsub('\\"', '"')),
      url = "https://vimeo.com/" .. id,
      thumbnail = "",
    }
    if #results >= 20 then break end
  end
  return results
end
