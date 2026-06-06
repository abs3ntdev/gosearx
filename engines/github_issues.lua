-- GitHub Issues / Pull Requests — rich search via /search/issues. The
-- config.gh_kind (issue|pr) appends the is:issue / is:pr qualifier so one file
-- backs both the issues and PRs engines.
-- @shortcut: ghi
-- @categories: it
-- @timeout: 6s

function request(query, params)
  local kind = (params.config and params.config.gh_kind) or "issue"
  local q = query
  if not q:find("is:" .. kind, 1, true) then
    q = q .. " is:" .. kind
  end
  params.url = "https://api.github.com/search/issues?" .. url.encode({
    q = q, sort = "updated", order = "desc",
    per_page = "15", page = tostring(params.pageno or 1),
  })
  params.headers["Accept"] = "application/vnd.github+json"
  params.headers["X-GitHub-Api-Version"] = "2022-11-28"
  local tok = params.config and params.config.token
  if tok and tok ~= "" then params.headers["Authorization"] = "Bearer " .. tok end
  return params
end

-- derive "owner/repo" from an issue's repository_url
local function repo_from_url(u)
  return (u or ""):match("repos/(.+)$") or ""
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, it in ipairs(data.items or {}) do
    local is_pr = it.pull_request ~= nil
    local state = it.state or "open"
    -- merged PRs report state=closed but have a merged_at timestamp
    if is_pr and type(it.pull_request) == "table" and it.pull_request.merged_at then
      state = "merged"
    end
    local labels = {}
    for _, l in ipairs(it.labels or {}) do
      if type(l) == "table" and l.name then labels[#labels + 1] = l.name end
    end
    results[#results + 1] = {
      type = "gh_issue",
      title = it.title or "",
      url = it.html_url,
      repo = repo_from_url(it.repository_url),
      number = it.number or 0,
      state = state,
      isPR = is_pr,
      draft = it.draft or false,
      author = (type(it.user) == "table" and it.user.login) or "",
      authorAvatar = (type(it.user) == "table" and it.user.avatar_url) or "",
      comments = it.comments or 0,
      labels = labels,
      created = it.created_at or "",
      body = (it.body or ""):sub(1, 280),
    }
  end
  return results
end
