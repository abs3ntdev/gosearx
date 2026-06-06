-- Reuters — news via the site's articles-by-search API (JSON-in-URL query).
-- @shortcut: reu
-- @categories: news
-- @timeout: 6s

local base = "https://www.reuters.com"
local per_page = 20

function request(query, params)
  local args = json.encode({
    keyword = query,
    offset = (params.pageno - 1) * per_page,
    orderby = "display_date:desc",
    size = per_page,
    website = "reuters",
  })
  params.url = base .. "/pf/api/v3/content/fetch/articles-by-search-v2?query=" .. url.escape(args)
  params.headers["Accept"] = "application/json"
  params.headers["User-Agent"] = "Mozilla/5.0 (gosearx engine)"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  local articles = (((data or {}).result or {}).articles)
  if type(articles) ~= "table" then return results end
  for _, a in ipairs(articles) do
    if a.canonical_url and a.web then
      local thumb = ""
      if type(a.thumbnail) == "table" then thumb = a.thumbnail.url or "" end
      results[#results + 1] = {
        title = a.web,
        url = base .. a.canonical_url,
        content = a.description or "",
        thumbnail = thumb,
        publishedDate = a.display_date or "",
      }
    end
  end
  return results
end
