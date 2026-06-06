import type { MainResult } from "../api";
import { highlightHTML } from "../highlight";

// MainResultCard renders a standard text/web result with a favicon, optional
// thumbnail image, and query-term highlighting — matching SearXNG's layout.
export function MainResultCard({
  result,
  query,
}: {
  result: MainResult;
  query: string;
}): React.JSX.Element {
  let host = result.url;
  try {
    host = new URL(result.url).host;
  } catch {
    /* keep raw url */
  }
  const favicon = `/favicon_proxy?host=${encodeURIComponent(host)}`;
  const thumb = result.thumbnail
    ? `/image_proxy?url=${encodeURIComponent(result.thumbnail)}`
    : "";

  // Highlight query terms in the description body only, never the title/link.
  const contentHTML = result.content ? highlightHTML(result.content, query) : "";

  return (
    <article className={thumb ? "result has-thumb" : "result"}>
      {thumb && (
        <a className="result-thumb" href={result.url} target="_blank" rel="noreferrer">
          <img src={thumb} alt="" loading="lazy" onError={hideImg} />
        </a>
      )}
      <div className="result-body">
        <div className="result-header">
          <img
            className="result-favicon"
            src={favicon}
            alt=""
            width={16}
            height={16}
            loading="lazy"
            onError={hideImg}
          />
          <a className="result-url" href={result.url} target="_blank" rel="noreferrer">
            {host}
          </a>
        </div>
        <h3 className="result-title">
          <a href={result.url} target="_blank" rel="noreferrer">
            {result.title}
          </a>
        </h3>
        {contentHTML && (
          <p className="result-content" dangerouslySetInnerHTML={{ __html: contentHTML }} />
        )}
        <div className="result-meta">
          {(result.engines ?? [result.engine]).map((e) => (
            <span className="badge" key={e}>
              {e}
            </span>
          ))}
          {result.publishedDate && <span className="result-date">{result.publishedDate}</span>}
        </div>
      </div>
    </article>
  );
}

function hideImg(e: React.SyntheticEvent<HTMLImageElement>) {
  (e.currentTarget as HTMLImageElement).style.visibility = "hidden";
}
