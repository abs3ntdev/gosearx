// Morse code converter — example of an in-process JavaScript plugin (goja).
//
// Backend: .mjs files run in a sandboxed, pure-Go JS runtime (no network, no
// filesystem). Define any of preSearch(ctx) / onResult(r) / postSearch(ctx) and
// an optional `keywords` array. Results use the same map schema as Lua/native
// plugins and are passed through result.FromMap.
//
// Usage: "morse hello world"  /  "unmorse .... .."

var keywords = ["morse", "unmorse"];

var TABLE = {
  a: ".-", b: "-...", c: "-.-.", d: "-..", e: ".", f: "..-.", g: "--.",
  h: "....", i: "..", j: ".---", k: "-.-", l: ".-..", m: "--", n: "-.",
  o: "---", p: ".--.", q: "--.-", r: ".-.", s: "...", t: "-", u: "..-",
  v: "...-", w: ".--", x: "-..-", y: "-.--", z: "--..",
  "0": "-----", "1": ".----", "2": "..---", "3": "...--", "4": "....-",
  "5": ".....", "6": "-....", "7": "--...", "8": "---..", "9": "----.",
};
var REV = {};
Object.keys(TABLE).forEach(function (k) { REV[TABLE[k]] = k; });

function encode(text) {
  return text.toLowerCase().split("").map(function (ch) {
    if (ch === " ") return "/";
    return TABLE[ch] || "";
  }).filter(Boolean).join(" ");
}

function decode(code) {
  return code.split(" ").map(function (sym) {
    if (sym === "/") return " ";
    return REV[sym] || "";
  }).join("");
}

function postSearch(ctx) {
  var q = ctx.query.trim();
  var sp = q.indexOf(" ");
  if (sp < 0) return [];
  var cmd = q.slice(0, sp).toLowerCase();
  var rest = q.slice(sp + 1);
  var out = cmd === "unmorse" ? decode(rest) : encode(rest);
  if (!out) return [];
  return [{ type: "answer", answer: out }];
}
