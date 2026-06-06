-- The Pirate Bay — torrents via the apibay JSON API.
-- @shortcut: tpb
-- @categories: files
-- @timeout: 5s

function request(query, params)
  params.url = "https://apibay.org/q.php?" .. url.encode({ q = query })
  params.headers["Accept"] = "application/json"
  return params
end

local function human_size(n)
  n = tonumber(n) or 0
  local units = { "B", "KB", "MB", "GB", "TB" }
  local i = 1
  while n >= 1024 and i < #units do n = n / 1024; i = i + 1 end
  return string.format("%.1f %s", n, units[i])
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  local data = json.decode(resp.text)
  if type(data) ~= "table" then return results end
  for _, r in ipairs(data) do
    -- apibay returns a single {id:"0",name:"No results..."} when empty
    if r.id and r.id ~= "0" and r.name then
      local magnet = "magnet:?xt=urn:btih:" .. (r.info_hash or "") .. "&dn=" .. url.escape(r.name)
      results[#results + 1] = {
        type = "torrent",
        title = r.name,
        url = "https://thepiratebay.org/description.php?id=" .. tostring(r.id),
        magnetLink = magnet,
        seeders = tonumber(r.seeders) or 0,
        leechers = tonumber(r.leechers) or 0,
        fileSize = human_size(r.size),
        files = tonumber(r.num_files) or 0,
      }
    end
  end
  return results
end
