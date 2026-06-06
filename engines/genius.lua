-- Genius — music/lyrics via the public API (no key).
-- @shortcut: gen
-- @categories: music
-- @timeout: 5s

function request(query, params)
  params.url = "https://genius.com/api/search/multi?" .. url.encode({ q = query, per_page = "5" })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" or type(data.response) ~= "table" then return results end
  for _, section in ipairs(data.response.sections or {}) do
    for _, hit in ipairs(section.hits or {}) do
      local r = hit.result or {}
      if r.url and (r.full_title or r.title) then
        results[#results + 1] = {
          title = r.full_title or r.title,
          url = r.url,
          content = (r.primary_artist and r.primary_artist.name) or "",
          thumbnail = r.song_art_image_thumbnail_url or r.header_image_thumbnail_url or "",
        }
      end
    end
  end
  return results
end
