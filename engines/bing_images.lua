-- Bing Images — Lua port of bing_images.py. Each result anchor has a JSON
-- blob in its `m` attribute with murl (full image) + turl (thumbnail).
-- @shortcut: bii
-- @categories: images
-- @timeout: 5s

function request(query, params)
  local first = (params.pageno - 1) * 35 + 1
  params.url = "https://www.bing.com/images/async?" .. url.encode({
    q = query, first = tostring(first), count = "35", async = "1",
  })
  params.headers["Accept"] = "text/html"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local dom = html.parse(resp.text)
  for _, a in ipairs(xpath.list(dom, '//a[@class="iusc"]')) do
    local m = xpath.attr(a, ".", "m")
    if m ~= "" then
      local meta = json.decode(m)
      if type(meta) == "table" and meta.murl then
        results[#results + 1] = {
          type = "image",
          title = meta.t or "",
          url = meta.purl or meta.murl,
          imgSrc = meta.murl,
          thumbnailSrc = meta.turl or meta.murl,
          source = "Bing",
        }
      end
    end
  end
  return results
end
