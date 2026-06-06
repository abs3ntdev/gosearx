-- calculator: evaluates simple arithmetic expressions and returns an answer.
-- Port of SearXNG's calculator plugin. Implemented as a self-contained
-- recursive-descent parser (the Lua sandbox forbids load()/eval).
-- @about: Parses and solves mathematical expressions.

-- Tokenizer: numbers, + - * / ( ) and ^.
local function tokenize(s)
  local toks, i, n = {}, 1, #s
  while i <= n do
    local c = s:sub(i, i)
    if c:match("%s") then
      i = i + 1
    elseif c:match("[%d.]") then
      local j = i
      while j <= n and s:sub(j, j):match("[%d.]") do j = j + 1 end
      toks[#toks + 1] = { t = "num", v = tonumber(s:sub(i, j - 1)) }
      i = j
    elseif c:match("[%+%-%*/%^%(%)]") then
      toks[#toks + 1] = { t = c }
      i = i + 1
    else
      return nil -- unsupported character -> not a calc expression
    end
  end
  return toks
end

-- Recursive-descent parser/evaluator with precedence: ^ > * / > + -.
local function evaluate(toks)
  local pos = 1
  local function peek() return toks[pos] end
  local function eat() local t = toks[pos]; pos = pos + 1; return t end

  local parseExpr
  local function parseAtom()
    local t = peek()
    if not t then return nil end
    if t.t == "(" then
      eat()
      local v = parseExpr()
      if not peek() or peek().t ~= ")" then return nil end
      eat()
      return v
    elseif t.t == "-" then
      eat()
      local v = parseAtom()
      if v == nil then return nil end
      return -v
    elseif t.t == "num" then
      eat()
      return t.v
    end
    return nil
  end

  local function parsePow()
    local base = parseAtom()
    if base == nil then return nil end
    if peek() and peek().t == "^" then
      eat()
      local exp = parsePow()
      if exp == nil then return nil end
      return base ^ exp
    end
    return base
  end

  local function parseTerm()
    local v = parsePow()
    if v == nil then return nil end
    while peek() and (peek().t == "*" or peek().t == "/") do
      local op = eat().t
      local rhs = parsePow()
      if rhs == nil then return nil end
      if op == "*" then v = v * rhs else v = v / rhs end
    end
    return v
  end

  parseExpr = function()
    local v = parseTerm()
    if v == nil then return nil end
    while peek() and (peek().t == "+" or peek().t == "-") do
      local op = eat().t
      local rhs = parseTerm()
      if rhs == nil then return nil end
      if op == "+" then v = v + rhs else v = v - rhs end
    end
    return v
  end

  local result = parseExpr()
  if pos <= #toks then return nil end -- trailing junk
  return result
end

function post_search(ctx)
  local q = ctx.query
  -- Only attempt if the query looks like math (has a digit and an operator).
  if not (q:match("%d") and q:match("[%+%-%*/%^]")) then
    return {}
  end
  local toks = tokenize(q)
  if not toks or #toks == 0 then return {} end
  local ok, val = pcall(evaluate, toks)
  if not ok or val == nil then return {} end

  -- Format: integers without decimals, else trim.
  local text
  if val == math.floor(val) then
    text = string.format("%d", val)
  else
    text = string.format("%.10g", val)
  end
  return { { type = "answer", answer = q .. " = " .. text } }
end
