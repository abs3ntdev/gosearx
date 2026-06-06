-- PubMed — scientific publications. Two-step: esearch for IDs, then esummary
-- for metadata (uses the engine http.get capability).
-- @shortcut: pub
-- @categories: science
-- @timeout: 8s

local eutils = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils"

function request(query, params)
  local retstart = (params.pageno - 1) * 10
  params.url = eutils .. "/esearch.fcgi?" .. url.encode({
    db = "pubmed", term = query, retstart = tostring(retstart), retmax = "10",
  })
  params.headers["Accept"] = "application/xml"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  -- collect PMIDs from the esearch XML
  local ids = {}
  for id in resp.text:gmatch("<Id>(%d+)</Id>") do ids[#ids + 1] = id end
  if #ids == 0 then return results end

  -- fetch summaries in one esummary call
  local sresp = http.get(eutils .. "/esummary.fcgi?" .. url.encode({
    db = "pubmed", id = table.concat(ids, ","), retmode = "json",
  }))
  if not sresp or sresp.status_code ~= 200 then
    -- fall back to bare links
    for _, id in ipairs(ids) do
      results[#results + 1] = { title = "PMID " .. id, url = "https://pubmed.ncbi.nlm.nih.gov/" .. id .. "/" }
    end
    return results
  end
  local data = json.decode(sresp.text)
  local result = (data or {}).result or {}
  for _, id in ipairs(ids) do
    local doc = result[id]
    if type(doc) == "table" then
      local authors = {}
      for _, a in ipairs(doc.authors or {}) do authors[#authors + 1] = a.name end
      results[#results + 1] = {
        type = "paper",
        title = doc.title or ("PMID " .. id),
        url = "https://pubmed.ncbi.nlm.nih.gov/" .. id .. "/",
        content = "",
        authors = authors,
        journal = doc.fulljournalname or "",
        publishedDate = doc.pubdate or "",
      }
    end
  end
  return results
end
