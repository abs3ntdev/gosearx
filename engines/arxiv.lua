-- arXiv — scientific papers via the official Atom API (no key).
-- @shortcut: arx
-- @categories: science
-- @timeout: 6s

function request(query, params)
  local start = (params.pageno - 1) * 10
  params.url = "https://export.arxiv.org/api/query?"
    .. url.encode({ search_query = "all:" .. query, start = tostring(start), max_results = "10" })
  params.headers["Accept"] = "application/atom+xml"
  return params
end

local function strip(s) return (s:gsub("%s+", " "):gsub("^%s+", ""):gsub("%s+$", "")) end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  for entry in resp.text:gmatch("<entry>(.-)</entry>") do
    local title = entry:match("<title>(.-)</title>")
    local link = entry:match('<id>(.-)</id>')
    local summary = entry:match("<summary>(.-)</summary>")
    local published = entry:match("<published>(.-)</published>")
    local pdf = entry:match('<link[^>]-title="pdf"[^>]-href="([^"]+)"')
    local authors = {}
    for a in entry:gmatch("<author>%s*<name>(.-)</name>") do
      authors[#authors + 1] = strip(a)
    end
    if title and link then
      results[#results + 1] = {
        type = "paper",
        title = strip(title),
        url = link,
        content = strip(summary or ""),
        authors = authors,
        publishedDate = published or "",
        pdfUrl = pdf or "",
        journal = "arXiv",
      }
    end
  end
  return results
end
