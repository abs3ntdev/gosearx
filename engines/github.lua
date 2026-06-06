-- GitHub repositories — rich first-class repo search via the REST API.
-- Emits gh_repo results (stars, language, license, topics, avatar). Supports an
-- optional API token (config.token) for 5000 req/hr and to avoid throttling.
-- Raw GitHub qualifiers in the query pass through (stars:>100, language:go, ...).
-- @shortcut: gh
-- @categories: it, repos
-- @timeout: 6s

local function auth(params)
  params.headers["Accept"] = "application/vnd.github+json"
  params.headers["X-GitHub-Api-Version"] = "2022-11-28"
  local tok = params.config and params.config.token
  if tok and tok ~= "" then params.headers["Authorization"] = "Bearer " .. tok end
end

function request(query, params)
  params.url = "https://api.github.com/search/repositories?" .. url.encode({
    q = query,
    sort = "stars",
    order = "desc",
    per_page = "15",
    page = tostring(params.pageno or 1),
  })
  auth(params)
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, it in ipairs(data.items or {}) do
    results[#results + 1] = {
      type = "gh_repo",
      fullName = it.full_name,
      url = it.html_url,
      description = it.description or "",
      language = it.language or "",
      stars = it.stargazers_count or 0,
      forks = it.forks_count or 0,
      openIssues = it.open_issues_count or 0,
      license = (type(it.license) == "table" and it.license.spdx_id) or "",
      updated = it.pushed_at or it.updated_at or "",
      topics = (type(it.topics) == "table" and #it.topics > 0) and it.topics or nil,
      ownerAvatar = (type(it.owner) == "table" and it.owner.avatar_url) or "",
      archived = it.archived or false,
    }
  end
  return results
end
