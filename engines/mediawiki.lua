-- MediaWiki — generic engine for any MediaWiki instance, configured via
-- settings.yml config.base_url (e.g. https://wiki.archlinux.org/). Port of the
-- common path of searx/engines/mediawiki.py using the opensearch API.
-- Use one settings.yml entry per wiki, all pointing at this engine file by
-- giving each its own config.base_url.
-- @shortcut: mw
-- @categories: general
-- @timeout: 5s

function request(query, params)
  local base = params.config.base_url or "https://en.wikipedia.org/"
  if base:sub(-1) ~= "/" then base = base .. "/" end
  params.url = base .. "w/api.php?" .. url.encode({
    action = "opensearch",
    format = "json",
    limit = "10",
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
  return results
end
