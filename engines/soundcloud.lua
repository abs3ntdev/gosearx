-- SoundCloud — music search via the public v2 API. Needs a client_id which
-- SoundCloud embeds in its web player; set it in settings.yml config.client_id.
-- Without one it falls back gracefully (no results).
-- @shortcut: sc
-- @categories: music
-- @timeout: 5s

function request(query, params)
  local cid = params.config and params.config.client_id or ""
  params.url = "https://api-v2.soundcloud.com/search?"
    .. url.encode({
      q = query,
      client_id = cid,
      limit = "20",
      offset = tostring((params.pageno - 1) * 20),
    })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data.collection or {}) do
    if r.permalink_url and r.title then
      results[#results + 1] = {
        title = r.title,
        url = r.permalink_url,
        content = (r.user and r.user.username or "") .. (r.description and (" — " .. r.description) or ""),
        thumbnail = r.artwork_url or "",
      }
    end
  end
  return results
end
