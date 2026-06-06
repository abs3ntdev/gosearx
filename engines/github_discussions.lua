-- GitHub Discussions search — via the GraphQL API (REST has no discussion
-- search). Requires a token (GraphQL always needs auth). The host fetch hits a
-- no-op; the real GraphQL POST happens in response() via http.post.
-- @shortcut: ghd
-- @categories: it, github
-- @timeout: 7s

local GQL = [[
query($q: String!) {
  search(query: $q, type: DISCUSSION, first: 15) {
    nodes {
      ... on Discussion {
        title url number bodyText
        createdAt isAnswered
        comments { totalCount }
        author { login }
        category { name }
        repository { nameWithOwner }
      }
    }
  }
}]]

function request(query, params)
  -- nothing to fetch up front; carry the token via a tiny self request.
  params.url = "https://api.github.com/meta"
  params.headers["Accept"] = "application/vnd.github+json"
  return params
end

function response(resp)
  local results = {}
  local tok = resp.config and resp.config.token
  if not tok or tok == "" then return results end -- GraphQL requires auth

  local body = json.encode({
    query = GQL,
    variables = { q = resp.query or "" },
  })
  local gr = http.post("https://api.github.com/graphql", body, {
    ["Authorization"] = "Bearer " .. tok,
    ["Content-Type"] = "application/json",
    ["Accept"] = "application/json",
  })
  if not gr or gr.status_code ~= 200 then return results end
  local data = json.decode(gr.text)
  local nodes = (((data or {}).data or {}).search or {}).nodes
  if type(nodes) ~= "table" then return results end

  for _, n in ipairs(nodes) do
    if n.url then
      results[#results + 1] = {
        type = "gh_discussion",
        title = n.title or "",
        url = n.url,
        repo = (type(n.repository) == "table" and n.repository.nameWithOwner) or "",
        number = n.number or 0,
        category = (type(n.category) == "table" and n.category.name) or "",
        author = (type(n.author) == "table" and n.author.login) or "",
        answered = n.isAnswered or false,
        comments = (type(n.comments) == "table" and n.comments.totalCount) or 0,
        created = n.createdAt or "",
        body = (n.bodyText or ""):sub(1, 280),
      }
    end
  end
  return results
end
