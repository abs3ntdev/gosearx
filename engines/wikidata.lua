-- Wikidata — entity search via wbsearchentities, then fetch the top entity's
-- claims/description for an infobox. (A pragmatic subset of SearXNG's SPARQL
-- engine: search + entity details, no full SPARQL.)
-- @shortcut: wd
-- @categories: general
-- @timeout: 6s

local api = "https://www.wikidata.org/w/api.php"

function request(query, params)
  params.url = api .. "?" .. url.encode({
    action = "wbsearchentities",
    search = query,
    language = "en",
    format = "json",
    limit = "5",
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" or type(data.search) ~= "table" then return results end

  local top = data.search[1]
  if not top then return results end

  -- list results
  for _, e in ipairs(data.search) do
    results[#results + 1] = {
      title = e.label or e.id,
      url = (e.concepturi or ("https://www.wikidata.org/wiki/" .. e.id)),
      content = e.description or "",
    }
  end

  -- infobox from the top entity's full record
  local er = http.get(api .. "?" .. url.encode({
    action = "wbgetentities", ids = top.id, languages = "en",
    props = "labels|descriptions|claims|sitelinks", format = "json",
  }), { ["Accept"] = "application/json" })
  local box = {
    type = "infobox",
    title = top.label or top.id,
    content = top.description or "",
    id = top.id,
    urls = { { title = "Wikidata", url = "https://www.wikidata.org/wiki/" .. top.id } },
  }
  if er and er.status_code == 200 then
    local ed = json.decode(er.text)
    local ent = ed and ed.entities and ed.entities[top.id]
    if type(ent) == "table" then
      -- Wikipedia sitelink as a friendlier primary URL
      local sl = ent.sitelinks
      if type(sl) == "table" and type(sl.enwiki) == "table" and sl.enwiki.title then
        box.urls[#box.urls + 1] = {
          title = "Wikipedia",
          url = "https://en.wikipedia.org/wiki/" .. (sl.enwiki.title:gsub(" ", "_")),
        }
      end
    end
  end
  results[#results + 1] = box
  return results
end
