-- Wordnik — dictionary definitions (scrapes the word page).
-- @shortcut: def
-- @categories: dictionaries
-- @timeout: 5s

function request(query, params)
  params.url = "https://www.wordnik.com/words/" .. url.escape(query)
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local dom = html.parse(resp.text)
  local word = resp.query or ""
  for _, li in ipairs(xpath.list(dom, '//div[contains(@class,"word-module")]//li')) do
    local def = tostring(li)
    if def and #def > 3 then
      results[#results + 1] = {
        type = "answer",
        answer = word .. ": " .. def,
      }
      if #results >= 5 then break end
    end
  end
  return results
end
