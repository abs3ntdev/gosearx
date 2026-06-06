-- GitHub Commits search — searches commit messages across GitHub.
-- @shortcut: ghx
-- @categories: it, github
-- @timeout: 6s

function request(query, params)
  params.url = "https://api.github.com/search/commits?" .. url.encode({
    q = query, sort = "committer-date", order = "desc",
    per_page = "15", page = tostring(params.pageno or 1),
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
    local commit = it.commit or {}
    local author = commit.author or {}
    -- first line of the commit message is the subject
    local msg = (commit.message or ""):match("^[^\n]*") or ""
    results[#results + 1] = {
      type = "gh_commit",
      sha = (it.sha or ""):sub(1, 8),
      url = it.html_url,
      repo = (type(it.repository) == "table" and it.repository.full_name) or "",
      message = msg,
      author = author.name or (type(it.author) == "table" and it.author.login) or "",
      date = author.date or "",
    }
  end
  return results
end
