-- Lingva — Google Translate frontend API (no key). Only triggers on an explicit
-- "translate <text> [to <lang>]" request, so it never fires on plain searches.
-- @shortcut: lv
-- @categories: translate
-- @timeout: 6s

local function parse(q, locale)
  q = (q or ""):gsub("^%s+", ""):gsub("%s+$", "")
  local text, to = q:match("^[Tt]ranslate%s+(.+)%s+to%s+(%a+)$")
  if text then return text, to:lower() end
  text = q:match("^[Tt]ranslate%s+(.+)$")
  if text then
    local target = "es"
    if locale and locale ~= "all" and #locale >= 2 then target = locale:sub(1, 2) end
    return text, target
  end
  return nil
end

function request(query, params)
  local text, to = parse(query, params.language)
  local base = (params.config and params.config.base_url) or "https://lingva.ml"
  if not text then
    params.url = base .. "/api/v1/en/es/x"
    return params
  end
  params.url = base .. "/api/v1/en/" .. to .. "/" .. url.escape(text)
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  if not parse(resp.query, "all") then return results end
  local data = json.decode(resp.text)
  if type(data) == "table" and data.translation and data.translation ~= "" then
    results[#results + 1] = { type = "answer", answer = "Lingva: " .. data.translation }
  end
  return results
end
