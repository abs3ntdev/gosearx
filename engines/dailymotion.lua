-- Dailymotion — videos, official public API (no key).
-- @shortcut: dm
-- @categories: videos
-- @timeout: 5s

local fields = "id,title,description,duration,url,thumbnail_360_url,owner.screenname"

function request(query, params)
  params.url = "https://api.dailymotion.com/videos?"
    .. url.encode({
      search = query,
      fields = fields,
      limit = "20",
      page = tostring(params.pageno or 1),
    })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data.list or {}) do
    results[#results + 1] = {
      type = "video",
      title = r.title or "",
      url = r.url or "",
      content = r.description or "",
      thumbnail = r["thumbnail_360_url"] or "",
    }
  end
  return results
end
