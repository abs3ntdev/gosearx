import { useCallback, useEffect, useRef, useState } from "react";
import {
  streamSearch,
  getConfig,
  type SearchResponse,
  type InstanceConfig,
} from "./api";
import { renderResult } from "./results/registry";
import { ChartCard } from "./results/ChartCard";
import { ImageGrid, QuoteCard } from "./results/extra";
import { GitHubResults } from "./results/GitHub";
import {
  PaperCard,
  TorrentCard,
  MapCard,
  VideoCard,
  CodeCard2,
  FileCard,
  KeyValueCard,
} from "./results/Templates";
import { SearchBox } from "./components/SearchBox";
import { Preferences } from "./components/Preferences";
import { AIAnswer } from "./components/AIAnswer";
import {
  Answers,
  CategoryTabs,
  Corrections,
  FilterBar,
  InfoboxCard,
  Pagination,
  SearchFormats,
  Suggestions,
} from "./components/Chrome";
import type { SearchFilters } from "./components/Chrome";

// applyFilters appends search operators (filetype:/site:/-site:) to the engine
// query. The major web engines (google, bing, ddg, brave, startpage) all honor
// these, so we keep filtering portable instead of per-engine API params.
function applyFilters(query: string, f: SearchFilters): string {
  const parts = [query.trim()];
  if (f.fileType) parts.push(`filetype:${f.fileType}`);
  if (f.siteInclude) parts.push(`site:${f.siteInclude.replace(/^https?:\/\//, "")}`);
  if (f.siteExclude) {
    const dom = f.siteExclude.replace(/^-/, "").replace(/^https?:\/\//, "");
    if (dom) parts.push(`-site:${dom}`);
  }
  return parts.filter(Boolean).join(" ");
}

// ---- URL <-> search-state syncing ------------------------------------------
// Searches live in the URL (?q=&categories=&pageno=) so the app is shareable,
// bookmarkable, usable as a browser search engine, and navigable with the
// browser's back/forward buttons. We canonicalize the results path to /search
// (SearXNG-compatible); the server serves the SPA for any path.

interface URLState {
  q: string;
  category: string;
  pageno: number;
}

// readURLState parses the current location's query params. `categories` is the
// SearXNG-style param name; `category` is accepted as an alias.
function readURLState(defaultCategory: string): URLState {
  const p = new URLSearchParams(window.location.search);
  const pageno = Math.max(1, parseInt(p.get("pageno") || "1", 10) || 1);
  return {
    q: p.get("q") ?? "",
    category: p.get("categories") || p.get("category") || defaultCategory,
    pageno,
  };
}

// syncURL writes the current search into the address bar at /search.
function syncURL(query: string, category: string, pageno: number, replace: boolean): void {
  const p = new URLSearchParams();
  p.set("q", query);
  if (category && category !== "general") p.set("categories", category);
  if (pageno > 1) p.set("pageno", String(pageno));
  const url = `/search?${p.toString()}`;
  if (replace) window.history.replaceState(null, "", url);
  else window.history.pushState(null, "", url);
}

// arrayKeys are the per-type result arrays that stream in and get concatenated.
const ARRAY_KEYS = [
  "results", "answers", "infoboxes", "images", "videos", "papers", "torrents",
  "maps", "codes", "files", "keyvalues", "quotes", "charts", "suggestions",
  "corrections", "ghRepos", "ghCode", "ghIssues", "ghUsers", "ghTopics",
  "ghCommits", "ghDiscussions", "charts", "quotes",
] as const;

// mergeChunk applies a streamed snapshot. The server already did cross-engine
// dedup + scoring, so for each per-type array present in the snapshot we REPLACE
// (not concat). The standalone `chart` event has no `engine` field, so it merges
// without clobbering the accumulated snapshot arrays.
function mergeChunk(
  prev: SearchResponse | null,
  chunk: Partial<SearchResponse> & { engine?: string; elapsed?: number },
  query: string,
  pageno: number,
): SearchResponse {
  const base: SearchResponse = prev ?? { query, pageno, results: [] };
  const next = { ...base } as unknown as Record<string, unknown>;
  const chunkRec = chunk as unknown as Record<string, unknown>;
  const isSnapshot = chunk.engine != null;

  for (const k of ARRAY_KEYS) {
    const incoming = chunkRec[k] as unknown[] | undefined;
    if (incoming === undefined) continue;
    if (isSnapshot) {
      next[k] = incoming; // full ranked snapshot replaces
    } else if (incoming.length) {
      // standalone event (e.g. chart): append
      const existing = (next[k] as unknown[] | undefined) ?? [];
      next[k] = [...existing, ...incoming];
    }
  }
  const result = next as unknown as SearchResponse;
  // De-dup charts/quotes by symbol (the chart event can race with snapshots).
  if (result.charts && result.charts.length > 1) {
    const seen = new Set<string>();
    result.charts = result.charts.filter((c) => {
      const k = c.symbol || c.title;
      if (seen.has(k)) return false;
      seen.add(k);
      return true;
    });
  }
  if (result.quotes && result.quotes.length > 1) {
    const seen = new Set<string>();
    result.quotes = result.quotes.filter((qt) => {
      if (seen.has(qt.symbol)) return false;
      seen.add(qt.symbol);
      return true;
    });
  }
  if (chunk.engine && chunk.elapsed != null) {
    result.timings = [...(result.timings ?? []), { engine: chunk.engine, total: chunk.elapsed }];
  }
  return result;
}

// hasRichResults reports whether the response carries any non-web results
// (GitHub cards, finance charts, answers, images, infoboxes) so we don't show
// "No results found" when those are present but the web results array is empty.
function hasRichResults(r: SearchResponse): boolean {
  return !!(
    r.ghRepos?.length ||
    r.ghCode?.length ||
    r.ghIssues?.length ||
    r.ghUsers?.length ||
    r.ghTopics?.length ||
    r.ghCommits?.length ||
    r.ghDiscussions?.length ||
    r.charts?.length ||
    r.answers?.length ||
    r.images?.length ||
    r.videos?.length ||
    r.papers?.length ||
    r.torrents?.length ||
    r.maps?.length ||
    r.codes?.length ||
    r.files?.length ||
    r.keyvalues?.length ||
    r.infoboxes?.length
  );
}

export function App(): React.JSX.Element {
  const [cfg, setCfg] = useState<InstanceConfig | null>(null);
  const [q, setQ] = useState("");
  const [submitted, setSubmitted] = useState(""); // last searched query
  const [category, setCategory] = useState("general");
  const [timeRange, setTimeRange] = useState("");
  const [language, setLanguage] = useState("all");
  const [safesearch, setSafesearch] = useState(0);
  const [filters, setFilters] = useState<SearchFilters>({
    fileType: "",
    siteInclude: "",
    siteExclude: "",
  });

  const [resp, setResp] = useState<SearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showPrefs, setShowPrefs] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  const streamAbortRef = useRef<(() => void) | null>(null);
  // Small LRU of completed responses for instant render on repeat queries.
  const cacheRef = useRef<Map<string, SearchResponse>>(new Map());
  // In-flight background pre-search streams (by cache key) so we don't dup them.
  const preStreamRef = useRef<Map<string, () => void>>(new Map());

  // Load instance config once.
  useEffect(() => {
    getConfig()
      .then((c) => {
        setCfg(c);
        setCategory(c.default_category || "general");
        setSafesearch(c.safe_search ?? 0);
      })
      .catch(() => setCfg(null));
  }, []);

  const runSearch = useCallback(
    async (overrides?: {
      q?: string;
      pageno?: number;
      category?: string;
      background?: boolean;
      // replaceHistory: replace the current URL instead of pushing a new entry
      // (used for filter-driven re-runs and the initial popstate restore).
      replaceHistory?: boolean;
      // skipHistory: don't touch the URL at all (used when restoring FROM the
      // URL via popstate / initial load, so we don't double-push).
      skipHistory?: boolean;
    }) => {
      const query = overrides?.q ?? q;
      if (!query.trim()) return;
      const page = overrides?.pageno ?? 1;
      const cat = overrides?.category ?? category;
      const background = overrides?.background ?? false;
      // Apply filter qualifiers (filetype:/site:/-site:) to the engine query
      // while keeping the displayed query clean.
      const engineQuery = applyFilters(query, filters);
      const cacheKey = `${cat}|${page}|${timeRange}|${language}|${safesearch}|${engineQuery}`;

      const setCache = (r: SearchResponse) => {
        cacheRef.current.set(cacheKey, r);
        if (cacheRef.current.size > 12) {
          cacheRef.current.delete(cacheRef.current.keys().next().value as string);
        }
      };

      // ---- Background pre-search: warm the cache only, never touch the UI. ----
      if (background) {
        if (cacheRef.current.has(cacheKey) || preStreamRef.current.has(cacheKey)) return;
        let acc: SearchResponse = { query, pageno: page, results: [] };
        const stop = streamSearch(
          { q: engineQuery, pageno: page, categories: [cat], timeRange, language, safesearch },
          (chunk) => {
            acc = mergeChunk(acc, chunk, query, page);
          },
          () => {},
          () => {
            preStreamRef.current.delete(cacheKey);
            setCache(acc);
          },
          () => preStreamRef.current.delete(cacheKey),
        );
        preStreamRef.current.set(cacheKey, stop);
        return;
      }

      // ---- Foreground (submitted) search ----
      // Reflect this search in the URL so it is shareable, bookmarkable,
      // usable as a browser search engine, and navigable with back/forward.
      if (!overrides?.skipHistory) {
        syncURL(query, cat, page, overrides?.replaceHistory ?? false);
      }

      // cancel any in-flight foreground stream
      abortRef.current?.abort();
      streamAbortRef.current?.();

      // instant render from cache (possibly warmed by pre-search)
      const cached = cacheRef.current.get(cacheKey);
      if (cached) {
        setSubmitted(query);
        setResp(cached);
        setLoading(false);
        window.scrollTo({ top: 0, behavior: "smooth" });
        return;
      }

      setLoading(true);
      setError(null);
      setSubmitted(query);
      setResp({ query, pageno: page, results: [] });
      window.scrollTo({ top: 0, behavior: "smooth" });

      streamAbortRef.current = streamSearch(
        { q: engineQuery, pageno: page, categories: [cat], timeRange, language, safesearch },
        (chunk) => setResp((prev) => mergeChunk(prev, chunk, query, page)),
        (u) =>
          setResp((prev) =>
            prev ? { ...prev, unresponsive: [...(prev.unresponsive ?? []), u] } : prev,
          ),
        () => {
          setLoading(false);
          setResp((prev) => {
            if (prev) setCache(prev);
            return prev;
          });
        },
        (e) => setError(e),
      );
    },
    [q, category, timeRange, language, safesearch, filters],
  );

  // Re-run when filters change (only if a search is active). These refine the
  // current search, so replace the URL rather than push a new history entry.
  useEffect(() => {
    if (submitted) runSearch({ q: submitted, pageno: 1, replaceHistory: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [timeRange, language, safesearch, filters]);

  // ---- URL-driven search: initial load + back/forward navigation. ----------
  // Runs once cfg is available (so we know the default category). Reads the
  // query from the URL (/search?q=… or /?q=…) and executes it, and restores
  // state when the user hits back/forward (popstate) — without re-pushing.
  useEffect(() => {
    if (!cfg) return;
    const applyFromURL = (replace: boolean) => {
      const st = readURLState(cfg.default_category || "general");
      if (!st.q.trim()) {
        // No query in the URL => show the landing page.
        setSubmitted("");
        setResp(null);
        return;
      }
      setQ(st.q);
      setCategory(st.category);
      runSearch({
        q: st.q,
        category: st.category,
        pageno: st.pageno,
        skipHistory: true,
        replaceHistory: replace,
      });
    };
    // Initial load: honor a ?q= the user arrived with (search-engine landing).
    applyFromURL(true);
    const onPop = () => applyFromURL(false);
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cfg]);

  // Debounced pre-search: once a search is active, as the user keeps typing and
  // pauses, warm the cache in the BACKGROUND (does not change the visible
  // results) so hitting Enter renders instantly.
  useEffect(() => {
    if (!submitted) return; // only after the first explicit search
    const query = q.trim();
    if (query.length < 3 || query === submitted) return;
    const t = setTimeout(() => runSearch({ q: query, background: true }), 400);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q]);

  function selectCategory(c: string) {
    setCategory(c);
    if (submitted) runSearch({ q: submitted, pageno: 1, category: c });
  }

  const hasSearched = submitted !== "" && resp !== null;
  const categories = cfg?.categories ?? ["general"];

  if (showPrefs) {
    return <Preferences onClose={() => setShowPrefs(false)} />;
  }

  // Landing (pre-search) view: centered search box.
  if (!hasSearched) {
    return (
      <div className="app landing">
        <h1 className="logo-big">{cfg?.instance_name ?? "gosearx"}</h1>
        <div className="landing-search">
          <SearchBox
            value={q}
            onChange={setQ}
            onSubmit={() => runSearch()}
            autocompleteMin={cfg?.autocomplete_min ?? 4}
            autocompleteEnabled={!!cfg?.autocomplete}
          />
        </div>
        <CategoryTabs categories={categories} active={category} onSelect={setCategory} />
        {error && <div className="error">⚠ {error}</div>}
        {loading && <div className="loading-indicator">Searching…</div>}
        <button className="prefs-link" onClick={() => setShowPrefs(true)}>
          ⚙ Preferences
        </button>
      </div>
    );
  }

  // Results view.
  return (
    <div className="app results-view">
      <header className="results-header">
        <a className="logo" href="/" onClick={(e) => { e.preventDefault(); location.reload(); }}>
          {cfg?.instance_name ?? "gosearx"}
        </a>
        <div className="header-search">
          <SearchBox
            value={q}
            onChange={setQ}
            onSubmit={() => runSearch()}
            autocompleteMin={cfg?.autocomplete_min ?? 4}
            autocompleteEnabled={!!cfg?.autocomplete}
          />
        </div>
        <button className="prefs-link" onClick={() => setShowPrefs(true)} title="Preferences">
          ⚙
        </button>
      </header>

      <CategoryTabs categories={categories} active={category} onSelect={selectCategory} />
      <FilterBar
        timeRange={timeRange}
        language={language}
        safesearch={safesearch}
        filters={filters}
        onTimeRange={setTimeRange}
        onLanguage={setLanguage}
        onSafesearch={setSafesearch}
        onFilters={setFilters}
      />

      {error && <div className="error">⚠ {error}</div>}

      <div className="results-layout">
        <main className="results-main">
          {loading && <div className="loading-bar" />}

          {cfg?.ai && submitted && (resp.results?.length ?? 0) > 0 && (
            <AIAnswer
              key={submitted}
              query={submitted}
              categories={[category]}
              auto={!!cfg.ai_auto}
            />
          )}

          {resp.charts?.map((c, i) => <ChartCard chart={c} key={`chart-${i}`} />)}
          {/* standalone quotes (those not already shown inside a chart) */}
          {(resp.quotes ?? [])
            .filter((qt) => !(resp.charts ?? []).some((c) => c.symbol === qt.symbol))
            .map((qt, i) => (
              <QuoteCard quote={qt} key={`quote-${i}`} />
            ))}
          <GitHubResults resp={resp} />
          {resp.answers && resp.answers.length > 0 && <Answers answers={resp.answers} />}
          {resp.corrections && resp.corrections.length > 0 && (
            <Corrections corrections={resp.corrections} onPick={(c) => { setQ(c); runSearch({ q: c }); }} />
          )}
          {resp.images && resp.images.length > 0 && <ImageGrid images={resp.images} />}

          {(resp.maps ?? []).map((m, i) => <MapCard place={m} key={`map-${i}`} />)}
          {(resp.papers ?? []).map((p, i) => <PaperCard paper={p} key={`paper-${i}`} />)}
          {(resp.videos ?? []).map((v, i) => <VideoCard video={v} key={`vid-${i}`} />)}
          {(resp.torrents ?? []).map((t, i) => <TorrentCard torrent={t} key={`tor-${i}`} />)}
          {(resp.codes ?? []).map((c, i) => <CodeCard2 code={c} key={`code-${i}`} />)}
          {(resp.files ?? []).map((f, i) => <FileCard file={f} key={`file-${i}`} />)}
          {(resp.keyvalues ?? []).map((k, i) => <KeyValueCard kv={k} key={`kv-${i}`} />)}

          {(resp.results?.length ?? 0) === 0 &&
            !loading &&
            !hasRichResults(resp) && <p className="empty">No results found.</p>}

          {(resp.results ?? []).map((r, i) => renderResult(r, i, submitted))}

          {(resp.results?.length ?? 0) > 0 && (
            <Pagination
              pageno={resp.pageno}
              hasResults={(resp.results?.length ?? 0) > 0}
              onPage={(p) => runSearch({ q: submitted, pageno: p })}
            />
          )}

          <div className="engine-timings">
            {resp.timings?.map((t) => (
              <span key={t.engine} className="badge">
                {t.engine} {(t.total * 1000).toFixed(0)}ms
              </span>
            ))}
            {resp.unresponsive?.map((u) => (
              <span key={u.engine} className="badge badge-error" title={u.reason}>
                {u.engine} failed
              </span>
            ))}
          </div>
        </main>

        <aside className="results-sidebar">
          {resp.infoboxes?.map((b, i) => <InfoboxCard box={b} key={i} />)}
          {resp.suggestions && resp.suggestions.length > 0 && (
            <Suggestions
              suggestions={resp.suggestions}
              onPick={(s) => { setQ(s); runSearch({ q: s }); }}
            />
          )}
          <SearchFormats query={submitted} />
        </aside>
      </div>
    </div>
  );
}
