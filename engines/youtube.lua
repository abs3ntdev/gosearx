-- YouTube (no API key) — scrapes the embedded ytInitialData JSON from the
-- results page. Port of searx/engines/youtube_noapi.py.
-- @shortcut: yt
-- @categories: videos
-- @timeout: 6s

function request(query, params)
  params.url = "https://www.youtube.com/results?" .. url.encode({ search_query = query })
  params.headers["Accept"] = "text/html"
  params.headers["Accept-Language"] = "en-US,en;q=0.9"
  params.cookies["CONSENT"] = "YES+"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  -- Extract the ytInitialData JSON blob.
  local blob = resp.text:match("ytInitialData%s*=%s*({.-});</script>")
    or resp.text:match("ytInitialData\"%]%s*=%s*({.-});")
  if not blob then return results end
  local data = json.decode(blob)
  if type(data) ~= "table" then return results end

  -- Walk to the search result contents.
  local ok, contents = pcall(function()
    return data.contents.twoColumnSearchResultsRenderer.primaryContents
      .sectionListRenderer.contents
  end)
  if not ok or type(contents) ~= "table" then return results end

  for _, section in ipairs(contents) do
    local items = section.itemSectionRenderer and section.itemSectionRenderer.contents or {}
    for _, it in ipairs(items) do
      local v = it.videoRenderer
      if v and v.videoId then
        local title = ""
        if v.title and v.title.runs and v.title.runs[1] then title = v.title.runs[1].text end
        results[#results + 1] = {
          type = "video",
          title = title,
          url = "https://www.youtube.com/watch?v=" .. v.videoId,
          thumbnail = "https://i.ytimg.com/vi/" .. v.videoId .. "/hqdefault.jpg",
          author = (v.ownerText and v.ownerText.runs and v.ownerText.runs[1] and v.ownerText.runs[1].text) or "",
          length = (v.lengthText and v.lengthText.simpleText) or "",
        }
      end
    end
  end
  return results
end
