-- unit_converter: converts between common units, e.g. "10 km in miles",
-- "100 f to c", "5 kg in lb". Port of SearXNG's unit_converter plugin (subset).
-- @about: Convert between common units of length, mass, and temperature.

-- conversion factors to a base unit per dimension.
local length = { -- base: meter
  mm = 0.001, cm = 0.01, m = 1, km = 1000,
  inch = 0.0254, ["in"] = 0.0254, ft = 0.3048, foot = 0.3048, feet = 0.3048,
  yd = 0.9144, yard = 0.9144, mi = 1609.344, mile = 1609.344, miles = 1609.344,
}
local mass = { -- base: gram
  mg = 0.001, g = 1, kg = 1000, t = 1000000,
  oz = 28.349523125, lb = 453.59237, lbs = 453.59237, pound = 453.59237,
}

local function convert_linear(tbl, v, from, to)
  if tbl[from] and tbl[to] then
    return v * tbl[from] / tbl[to]
  end
  return nil
end

-- temperature needs offset formulas, handled separately.
local function convert_temp(v, from, to)
  local c
  if from == "c" or from == "celsius" then c = v
  elseif from == "f" or from == "fahrenheit" then c = (v - 32) * 5 / 9
  elseif from == "k" or from == "kelvin" then c = v - 273.15
  else return nil end

  if to == "c" or to == "celsius" then return c
  elseif to == "f" or to == "fahrenheit" then return c * 9 / 5 + 32
  elseif to == "k" or to == "kelvin" then return c + 273.15
  else return nil end
end

local function fmt(n)
  if n == math.floor(n) then return string.format("%d", n) end
  return string.format("%.4g", n)
end

function post_search(ctx)
  -- pattern: <number> <unit> (in|to) <unit>
  local num, from, to = ctx.query:match("^%s*([%-%d%.]+)%s*(%a+)%s+to%s+(%a+)%s*$")
  if not num then
    num, from, to = ctx.query:match("^%s*([%-%d%.]+)%s*(%a+)%s+in%s+(%a+)%s*$")
  end
  if not num then return {} end

  local v = tonumber(num)
  if not v then return {} end
  from, to = from:lower(), to:lower()

  local result =
    convert_linear(length, v, from, to)
    or convert_linear(mass, v, from, to)
    or convert_temp(v, from, to)

  if result == nil then return {} end
  return { { type = "answer", answer = num .. " " .. from .. " = " .. fmt(result) .. " " .. to } }
end
