-- PDBe — Protein Data Bank Europe, science search via the SOLR API (no key).
-- @shortcut: pdb
-- @categories: science
-- @timeout: 6s

function request(query, params)
  params.url = "https://www.ebi.ac.uk/pdbe/search/pdb/select?" .. url.encode({
    q = query,
    wt = "json",
    rows = "20",
    start = tostring((params.pageno - 1) * 20),
    fl = "pdb_id,title,organism_scientific_name",
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  local docs = (((data or {}).response or {}).docs) or {}
  local seen = {}
  for _, d in ipairs(docs) do
    local id = d.pdb_id
    if id and not seen[id] then
      seen[id] = true
      results[#results + 1] = {
        title = (d.title or id) .. " (" .. id .. ")",
        url = "https://www.ebi.ac.uk/pdbe/entry/pdb/" .. id,
        content = type(d.organism_scientific_name) == "table"
          and table.concat(d.organism_scientific_name, ", ") or "",
        thumbnail = "https://www.ebi.ac.uk/pdbe/static/entry/" .. id .. "_deposited_chain_front_image-200x200.png",
      }
    end
  end
  return results
end
