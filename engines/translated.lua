-- MyMemory translate — only triggers on an explicit translate request so it
-- never fires on ordinary searches. Forms:
--   "translate <text> to <lang>"
--   "translate <text>"            (-> from en to the UI language, default es)
-- @shortcut: tl
-- @categories: general, translate
-- @timeout: 5s

-- parse returns text, from, to (or nil if not a translate request).
local function parse(q, locale)
  q = (q or ""):gsub("^%s+", ""):gsub("%s+$", "")
  local text, to = q:match("^[Tt]ranslate%s+(.+)%s+to%s+(%a+)$")
  if text then return text, "en", to:lower() end
  text = q:match("^[Tt]ranslate%s+(.+)$")
  if text then
    local target = "es"
    if locale and locale ~= "all" and #locale >= 2 then target = locale:sub(1, 2) end
    return text, "en", target
  end
  return nil
end

function request(query, params)
  local text, from, to = parse(query, params.language)
  if not text then
    params.url = "https://api.mymemory.translated.net/get?q=x&langpair=en|es"
    return params
  end
  params.url = "https://api.mymemory.translated.net/get?"
    .. url.encode({ q = text, langpair = from .. "|" .. to })
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  if not parse(resp.query, "all") then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" or type(data.responseData) ~= "table" then return results end
  local t = data.responseData.translatedText
  if t and t ~= "" then
    results[#results + 1] = { type = "answer", answer = "Translation: " .. t }
  end
  return results
end
