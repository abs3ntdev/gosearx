-- GitHub Users / Organizations search.
-- @shortcut: ghu
-- @categories: it
-- @timeout: 6s

function request(query, params)
  params.url = "https://api.github.com/search/users?" .. url.encode({
    q = query, per_page = "15", page = tostring(params.pageno or 1),
  })
  params.headers["Accept"] = "application/vnd.github+json"
  params.headers["X-GitHub-Api-Version"] = "2022-11-28"
  local tok = params.config and params.config.token
  if tok and tok ~= "" then params.headers["Authorization"] = "Bearer " .. tok end
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, it in ipairs(data.items or {}) do
    results[#results + 1] = {
      type = "gh_user",
      login = it.login,
      url = it.html_url,
      avatar = it.avatar_url or "",
      isOrg = (it.type == "Organization"),
    }
  end
  return results
end
