-- Startpage (web) — Lua port of searx/engines/startpage.py. Startpage requires
-- an `sc` token scraped from its home page form, then a POST search with a
-- region cookie. We do the whole two-step flow inside response() using the
-- engine http.get/http.post capability (the initial host fetch grabs the form).
-- @shortcut: sp
-- @categories: general, web
-- @timeout: 8s

local base = "https://www.startpage.com"

function request(query, params)
  -- The host fetch loads the home page so response() can scrape the sc token
  -- (and the shared cookie jar keeps the session for the follow-up POST).
  params.url = base .. "/"
  params.headers["Accept"] = "text/html"
  params.headers["Accept-Language"] = "en-US,en;q=0.9"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  if resp.url:find("/sp/captcha") then return results end

  -- scrape the sc token from the search form
  local dom = html.parse(resp.text)
  local sc = xpath.attr(dom, '//form[@id="search"]//input[@name="sc"]', "value")
  if sc == "" then
    sc = resp.text:match('name="sc"%s+value="([^"]+)"') or ""
  end
  if sc == "" then return results end

  local form = url.encode({
    query = resp.query or "",
    cat = "web",
    t = "device",
    sc = sc,
    abp = "1",
    abe = "1",
    abd = "1",
    qsr = "all",
  })
  local sresp = http.post(base .. "/sp/search", form, {
    ["Content-Type"] = "application/x-www-form-urlencoded",
    ["Origin"] = base,
    ["Referer"] = base .. "/",
    ["Accept"] = "text/html",
  })
  if not sresp or sresp.status_code ~= 200 then return results end

  local sdom = html.parse(sresp.text)
  -- Result anchors carry the stable classes "result-title result-link"; the
  -- snippet is a sibling paragraph. Target the anchor directly (Startpage's
  -- container divs use obfuscated css-* class names).
  -- Each result is a list item containing the result-link anchor and a
  -- <p class="description ...">. Iterate the anchors and pull the description
  -- from the nearest ancestor that also holds it.
  for _, a in ipairs(xpath.list(sdom, '//a[contains(@class,"result-link")]')) do
    local link = xpath.attr(a, ".", "href")
    local title = xpath.text(a, ".//h2 | .//h3")
    if title == "" then title = tostring(a) end
    if title:find("{") or title:find("css-") then title = "" end
    -- description lives as a sibling/descendant of the anchor's ancestor list item
    local content = xpath.text(a, 'ancestor::li[1]//p[contains(@class,"description")]')
    if content == "" then
      content = xpath.text(a, 'ancestor::div[contains(@class,"result")][1]//p[contains(@class,"description")]')
    end
    if link ~= "" and link:sub(1, 4) == "http" and title ~= "" then
      results[#results + 1] = { url = link, title = title, content = content }
    end
  end
  return results
end
