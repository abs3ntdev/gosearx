-- Semantic Scholar — scientific papers via the public Graph API (no key).
-- @shortcut: se
-- @categories: science
-- @timeout: 6s

function request(query, params)
  local offset = (params.pageno - 1) * 10
  params.url = "https://api.semanticscholar.org/graph/v1/paper/search?"
    .. url.encode({
      query = query,
      offset = tostring(offset),
      limit = "10",
      fields = "title,abstract,url,year,authors",
    })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, p in ipairs(data.data or {}) do
    local authors = {}
    for _, a in ipairs(p.authors or {}) do authors[#authors + 1] = a.name end
    results[#results + 1] = {
      type = "paper",
      title = p.title or "",
      url = p.url or "",
      content = p.abstract or "",
      authors = authors,
      publishedDate = p.year and tostring(p.year) or "",
      journal = "Semantic Scholar",
    }
  end
  return results
end
