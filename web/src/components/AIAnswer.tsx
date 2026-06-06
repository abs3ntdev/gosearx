// AIAnswer renders an LLM-synthesized, cited summary above the search results.
// It streams tokens from /api/answer and linkifies inline [n] citations to the
// corresponding source.
import { useCallback, useEffect, useRef, useState } from "react";
import { askAI, type AICitation } from "../api";

// linkifyCitations splits answer text on [n] markers and turns each into a
// superscript link to the matching source URL.
function renderAnswer(text: string, cites: AICitation[]): React.ReactNode[] {
  const out: React.ReactNode[] = [];
  const re = /\[(\d+)\]/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let key = 0;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const n = Number(m[1]);
    const cite = cites[n - 1];
    if (cite) {
      out.push(
        <a
          key={`c${key++}`}
          className="ai-cite"
          href={cite.url}
          target="_blank"
          rel="noreferrer"
          title={cite.title}
        >
          {n}
        </a>,
      );
    } else {
      out.push(m[0]);
    }
    last = re.lastIndex;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

export function AIAnswer({
  query,
  categories,
  auto,
}: {
  query: string;
  categories: string[];
  auto: boolean;
}): React.JSX.Element | null {
  const [open, setOpen] = useState(auto);
  const [answer, setAnswer] = useState("");
  const [cites, setCites] = useState<AICitation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<(() => void) | null>(null);
  const ranFor = useRef<string>("");

  const run = useCallback(() => {
    abortRef.current?.();
    setAnswer("");
    setCites([]);
    setError(null);
    setLoading(true);
    ranFor.current = query;
    abortRef.current = askAI(
      query,
      categories,
      (c) => setCites(c),
      (delta) => setAnswer((prev) => prev + delta),
      () => setLoading(false),
      (e) => {
        setError(e);
        setLoading(false);
      },
    );
  }, [query, categories]);

  // Auto-run once per query when configured; reset panel on new query.
  useEffect(() => {
    setAnswer("");
    setCites([]);
    setError(null);
    setLoading(false);
    if (auto && query && ranFor.current !== query) {
      setOpen(true);
      run();
    }
    return () => abortRef.current?.();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query]);

  if (!query) return null;

  if (!open) {
    return (
      <div className="ai-answer collapsed">
        <button className="ai-ask-btn" onClick={() => { setOpen(true); run(); }}>
          ✨ Ask AI to summarize
        </button>
      </div>
    );
  }

  return (
    <div className="ai-answer">
      <div className="ai-head">
        <span className="ai-badge">✨ AI summary</span>
        {loading && <span className="ai-status">thinking…</span>}
        {!loading && !error && answer && (
          <button className="ai-regen" onClick={run} title="Regenerate">
            ↻
          </button>
        )}
      </div>
      {error ? (
        <div className="ai-error">{error}</div>
      ) : (
        <>
          <p className="ai-body">
            {renderAnswer(answer, cites)}
            {loading && <span className="ai-caret" />}
          </p>
          {cites.length > 0 && (
            <ol className="ai-sources">
              {cites.map((c, i) => (
                <li key={i}>
                  <a href={c.url} target="_blank" rel="noreferrer">
                    {c.title || c.url}
                  </a>
                </li>
              ))}
            </ol>
          )}
        </>
      )}
      <div className="ai-disclaimer">AI-generated from the sources above — verify important facts.</div>
    </div>
  );
}
