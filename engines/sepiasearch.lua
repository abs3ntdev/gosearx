-- SepiaSearch — PeerTube federated video search (no key).
-- @shortcut: sep
-- @categories: videos
-- @timeout: 6s

function request(query, params)
  local start = (params.pageno - 1) * 10
  params.url = "https://sepiasearch.org/api/v1/search/videos?" .. url.encode({
    search = query, start = tostring(start), count = "10", sort = "-match",
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, v in ipairs(data.data or {}) do
    local thumb = v.thumbnailUrl or ""
    if thumb ~= "" and thumb:sub(1, 4) ~= "http" then
      thumb = (v.account and v.account.host and ("https://" .. v.account.host) or "") .. thumb
    end
    results[#results + 1] = {
      type = "video",
      title = v.name or "",
      url = v.url or "",
      content = (v.description or ""),
      thumbnail = thumb,
      author = (v.account and v.account.displayName) or "",
    }
  end
  return results
end
