-- Wikipedia engine — opensearch for the result list, then a REST summary
-- fetch (via the engine http.get capability) for a rich infobox with extract
-- text and a thumbnail image. Mirrors SearXNG's wikipedia infobox.
-- @shortcut: wp
-- @categories: general
-- @timeout: 6s

local api = "https://en.wikipedia.org/w/api.php"
local rest = "https://en.wikipedia.org/api/rest_v1/page/summary/"

function request(query, params)
  params.url = api .. "?" .. url.encode({
    action = "opensearch",
    format = "json",
    limit = "8",
    search = query,
  })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end

  local titles = data[2] or {}
  local descriptions = data[3] or {}
  local urls = data[4] or {}

  for i = 1, #titles do
    results[#results + 1] = {
      title = titles[i],
      url = urls[i] or "",
      content = descriptions[i] or "",
    }
  end

  -- Only show an infobox when the top hit genuinely matches the query, so loose
  -- fuzzy matches (e.g. "voo stock" -> "Ronald R. Van Stockum", "eth price" ->
  -- "Eva Price") don't produce a misleading panel.
  --
  -- Rules: ignore common/generic words ("stock", "price", "news", …); require a
  -- WHOLE-WORD match (not substring, so "stock" must not match "Stockum"); and
  -- require that EVERY meaningful query term is present in the title as a word.
  local stop = {
    price = true, prices = true, news = true, weather = true, stock = true,
    stocks = true, share = true, shares = true, quote = true, ticker = true,
    the = true, a = true, an = true, of = true, ["and"] = true, ["or"] = true,
    vs = true, review = true, reviews = true, meaning = true, define = true,
    what = true, who = true, is = true, are = true, how = true, why = true,
    com = true, org = true, www = true, near = true, me = true, today = true,
  }
  -- build the set of whole words in the title (lowercased, alnum tokens)
  local function relevant(title, query)
    local words = {}
    for w in title:lower():gmatch("%a+") do words[w] = true end
    local meaningful, hits = 0, 0
    for w in query:lower():gmatch("%a+") do
      if #w >= 3 and not stop[w] then
        meaningful = meaningful + 1
        if words[w] then hits = hits + 1 end
      end
    end
    -- need at least one meaningful term, and ALL of them must match as words
    return meaningful > 0 and hits == meaningful
  end

  -- Build a rich infobox for the top hit via the REST summary endpoint.
  if titles[1] and relevant(titles[1], resp.query or "") then
    local page = titles[1]:gsub(" ", "_")
    local sresp = http.get(rest .. url.escape(page))
    local box = {
      type = "infobox",
      title = titles[1],
      content = descriptions[1] or "",
      urls = { { title = "Wikipedia", url = urls[1] or "" } },
    }
    if sresp and sresp.status_code == 200 then
      local s = json.decode(sresp.text)
      if type(s) == "table" then
        if s.extract and s.extract ~= "" then
          box.content = s.extract
        end
        if type(s.thumbnail) == "table" and s.thumbnail.source then
          box.imgSrc = s.thumbnail.source
        end
        if type(s.content_urls) == "table"
           and type(s.content_urls.desktop) == "table"
           and s.content_urls.desktop.page then
          box.urls = { { title = "Wikipedia", url = s.content_urls.desktop.page } }
        end
        -- a couple of useful attributes when present
        local attrs = {}
        if s.description and s.description ~= "" then
          attrs[#attrs + 1] = { label = "About", value = s.description }
        end
        if #attrs > 0 then box.attributes = attrs end
      end
    end
    results[#results + 1] = box
  end

  return results
end
