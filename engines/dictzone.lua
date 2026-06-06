-- DictZone — dictionary/translation, scrapes the result table.
-- @shortcut: dc
-- @categories: dictionaries
-- @timeout: 5s

function request(query, params)
  -- default en->hu path like SearXNG's default; en->es is more broadly useful
  params.url = "https://dictzone.com/english-spanish-dictionary/" .. url.escape(query)
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local dom = html.parse(resp.text)
  for _, tr in ipairs(xpath.list(dom, '//table[@class="r"]//tr')) do
    local from = xpath.text(tr, "./td[1]")
    local to = xpath.text(tr, "./td[2]")
    -- skip the header row and empties
    if from ~= "" and to ~= "" and from ~= "English" then
      results[#results + 1] = { type = "answer", answer = from .. " → " .. to }
      if #results >= 6 then break end
    end
  end
  return results
end
