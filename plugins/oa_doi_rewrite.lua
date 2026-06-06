-- oa_doi_rewrite: rewrites DOI links to an open-access resolver so paywalled
-- papers route through an OA gateway. Port of SearXNG's oa_doi_rewrite.
-- @about: Avoid paywalls by redirecting DOI links to an open-access resolver.

local oa_resolver = "https://oadoi.org/"

function on_result(result)
  local u = result.url or ""
  -- match doi.org links and rewrite to the OA resolver
  local doi = u:match("https?://[^/]*doi%.org/(.+)$")
  if doi then
    result.url = oa_resolver .. doi
  end
  return true
end
