-- Mastodon — searches accounts (default) or hashtags on a Mastodon instance.
-- The type/base_url come from config so one file backs users + hashtags.
-- @shortcut: mau
-- @categories: social media
-- @timeout: 5s

function request(query, params)
  local base = (params.config and params.config.base_url) or "https://mastodon.social"
  local mtype = (params.config and params.config.mastodon_type) or "accounts"
  params.url = base .. "/api/v2/search?" .. url.encode({
    q = query, type = mtype, resolve = "false", limit = "20",
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, a in ipairs(data.accounts or {}) do
    results[#results + 1] = {
      title = (a.display_name ~= "" and a.display_name or a.username) .. " (@" .. a.acct .. ")",
      url = a.url,
      content = (a.note or ""):gsub("<[^>]*>", ""),
      thumbnail = a.avatar or "",
    }
  end
  for _, h in ipairs(data.hashtags or {}) do
    results[#results + 1] = { title = "#" .. h.name, url = h.url, content = "" }
  end
  return results
end
