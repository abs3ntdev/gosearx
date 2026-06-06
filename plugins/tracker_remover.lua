-- tracker_remover: strips common tracking query parameters from result URLs.
-- Port of SearXNG's tracker_url_remover (on_result hook).
-- @about: Removes tracking arguments (utm_*, fbclid, gclid, …) from URLs.

local tracker_params = {
  ["utm_source"] = true, ["utm_medium"] = true, ["utm_campaign"] = true,
  ["utm_term"] = true, ["utm_content"] = true, ["utm_id"] = true,
  ["fbclid"] = true, ["gclid"] = true, ["dclid"] = true,
  ["msclkid"] = true, ["mc_eid"] = true, ["igshid"] = true,
  ["yclid"] = true, ["_openstat"] = true,
}

-- Split "a=1&b=2" -> filtered query string.
local function clean_query(qs)
  local kept = {}
  for pair in qs:gmatch("[^&]+") do
    local key = pair:match("^([^=]+)")
    if key and not tracker_params[key] then
      kept[#kept + 1] = pair
    end
  end
  return table.concat(kept, "&")
end

function on_result(result)
  local url = result.url or ""
  local base, qs = url:match("^(.-)%?(.*)$")
  if base and qs then
    local cleaned = clean_query(qs)
    if cleaned ~= "" then
      result.url = base .. "?" .. cleaned
    else
      result.url = base
    end
  end
  return true -- keep the (possibly cleaned) result
end
