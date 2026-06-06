-- GitHub Code search — finds matching files with code fragments.
-- NOTE: GitHub's code search REST endpoint REQUIRES authentication, so set
-- config.token in settings.yml or this engine returns nothing.
-- @shortcut: ghc
-- @categories: it
-- @timeout: 7s

function request(query, params)
  params.url = "https://api.github.com/search/code?" .. url.encode({
    q = query, per_page = "15", page = tostring(params.pageno or 1),
  })
  params.headers["Accept"] = "application/vnd.github.text-match+json"
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
    local frags = {}
    for _, tm in ipairs(it.text_matches or {}) do
      if tm.fragment and tm.fragment ~= "" then frags[#frags + 1] = tm.fragment end
    end
    results[#results + 1] = {
      type = "gh_code",
      path = it.path or it.name,
      repo = (type(it.repository) == "table" and it.repository.full_name) or "",
      url = it.html_url,
      fragments = frags,
    }
  end
  return results
end
