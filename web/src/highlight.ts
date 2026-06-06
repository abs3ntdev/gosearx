// highlight.ts wraps occurrences of the query terms in <mark> tags within a
// snippet, mirroring SearXNG's term highlighting. The snippet may already be
// sanitized HTML, so we only highlight inside text nodes (never inside tags or
// attribute values) to avoid corrupting markup.

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

// buildTermRegex builds a case-insensitive regex matching any query term.
function buildTermRegex(query: string): RegExp | null {
  const terms = query
    .split(/\s+/)
    .map((t) => t.replace(/^[!:]+/, "").trim()) // drop bang/category prefixes
    .filter((t) => t.length >= 2)
    .map(escapeRegExp);
  if (terms.length === 0) return null;
  // longer terms first so multi-word fragments win
  terms.sort((a, b) => b.length - a.length);
  return new RegExp(`(${terms.join("|")})`, "gi");
}

// highlightHTML returns the snippet HTML with query terms wrapped in <mark>,
// highlighting only the text content between tags.
export function highlightHTML(htmlContent: string, query: string): string {
  const re = buildTermRegex(query);
  if (!re) return htmlContent;

  // Split on tags so we only touch text segments (odd indices are tags).
  const parts = htmlContent.split(/(<[^>]+>)/g);
  return parts
    .map((part) => {
      if (part.startsWith("<")) return part; // a tag, leave untouched
      return part.replace(re, "<mark>$1</mark>");
    })
    .join("");
}
