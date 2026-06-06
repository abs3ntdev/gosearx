-- Currency converter — "100 usd to eur" style, via the open exchangerate API.
-- Self-gates on the "<amount> <cur> to <cur>" pattern, so it shows nothing for
-- ordinary searches even though it can run in the general category.
-- @shortcut: cc
-- @categories: general, currency
-- @timeout: 5s

local cur3 = "[A-Za-z][A-Za-z][A-Za-z]"

function request(query, params)
  -- parse "<amount> <from> to <to>"
  local amount, from, to = query:match("^%s*([%d%.]+)%s+(" .. cur3 .. ")%s+to%s+(" .. cur3 .. ")%s*$")
  if not amount then
    amount, from, to = query:match("^%s*([%d%.]+)%s+(" .. cur3 .. ")%s+in%s+(" .. cur3 .. ")%s*$")
  end
  params.config = params.config or {}
  if not amount then
    -- not a currency query: hit a harmless URL and emit nothing
    params.url = "https://open.er-api.com/v6/latest/USD"
    params.config._skip = "1"
    return params
  end
  params.config._amount = amount
  params.config._from = from:upper()
  params.config._to = to:upper()
  params.url = "https://open.er-api.com/v6/latest/" .. from:upper()
  params.headers["Accept"] = "application/json"
  return params
end

function response(resp)
  local results = {}
  if resp.status_code ~= 200 then return results end
  -- We can't read params.config here; re-derive from resp.query.
  local q = resp.query or ""
  local amount, from, to = q:match("^%s*([%d%.]+)%s+(" .. cur3 .. ")%s+to%s+(" .. cur3 .. ")%s*$")
  if not amount then
    amount, from, to = q:match("^%s*([%d%.]+)%s+(" .. cur3 .. ")%s+in%s+(" .. cur3 .. ")%s*$")
  end
  if not amount then return results end
  to = to:upper()
  local data = json.decode(resp.text)
  if type(data) ~= "table" or type(data.rates) ~= "table" then return results end
  local rate = data.rates[to]
  if not rate then return results end
  local converted = tonumber(amount) * rate
  results[#results + 1] = {
    type = "answer",
    answer = string.format("%s %s = %.4g %s", amount, from:upper(), converted, to),
  }
  return results
end
