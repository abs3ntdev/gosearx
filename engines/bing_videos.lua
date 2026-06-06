-- Bing Videos — Lua port of bing_videos.py. Each tile has a JSON blob in the
-- `vrhm` attribute of div.vrhdata.
-- @shortcut: biv
-- @categories: videos
-- @timeout: 5s

function request(query, params)
  local first = (params.pageno - 1) * 35 + 1
  params.url = "https://www.bing.com/videos/asyncv2?" .. url.encode({
    q = query, first = tostring(first), count = "35", async = "content",
  })
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local dom = html.parse(resp.text)
  for _, d in ipairs(xpath.list(dom, '//div[@class="vrhdata"]')) do
    local m = xpath.attr(d, ".", "vrhm")
    if m ~= "" then
      local meta = json.decode(m)
      if type(meta) == "table" and meta.murl then
        results[#results + 1] = {
          type = "video",
          title = meta.vt or "",
          url = meta.murl,
          thumbnail = meta.thumbnailUrl or "",
        }
      end
    end
  end
  return results
end
