// Hacker News (Algolia API) — example of a JavaScript ENGINE (goja, in-process).
//
// @shortcut: hn
// @categories: it, news
// @timeout: 5s
//
// Backend: .mjs engine files run in a sandboxed pure-Go JS runtime. Define
// request(query, params) and response(resp); the HOST performs the HTTP fetch
// between them (engines never do I/O themselves), exactly like the Lua tier.
// Helpers available: url.encode/escape/unescape, base64, and native JSON.

function request(query, params) {
  var page = (params.pageno || 1) - 1;
  params.url =
    "https://hn.algolia.com/api/v1/search?" +
    url.encode({ query: query, page: String(page), tags: "story" });
  return params;
}

function response(resp) {
  var data = JSON.parse(resp.text);
  var hits = data.hits || [];
  return hits
    .filter(function (h) { return h.title; })
    .map(function (h) {
      var link = h.url || "https://news.ycombinator.com/item?id=" + h.objectID;
      var bits = [];
      if (h.points != null) bits.push(h.points + " points");
      if (h.num_comments != null) bits.push(h.num_comments + " comments");
      if (h.author) bits.push("by " + h.author);
      return {
        title: h.title,
        url: link,
        content: bits.join(" · "),
        publishedDate: h.created_at || "",
      };
    });
}
