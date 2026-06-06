// API types mirror the Go result union (internal/result) and the /api/search
// response (internal/server). Keep these in sync with the backend structs.

export type ResultType =
  | "main"
  | "image"
  | "video"
  | "answer"
  | "suggestion"
  | "correction"
  | "infobox"
  | "quote"
  | "chart";

export interface MainResult {
  type: ResultType;
  engine: string;
  title: string;
  url: string;
  content?: string;
  thumbnail?: string;
  publishedDate?: string;
  engines?: string[];
  score?: number;
}

export interface Answer {
  engine: string;
  answer: string;
  url?: string;
}

export interface ImageResult {
  type: ResultType;
  engine: string;
  title: string;
  url: string;
  imgSrc: string;
  thumbnailSrc: string;
  resolution?: string;
  source?: string;
}

export interface Quote {
  type: ResultType;
  engine: string;
  symbol: string;
  name?: string;
  currency?: string;
  price: number;
  change?: number;
  changePct?: number;
  url?: string;
}

export interface Candle {
  t: number;
  o: number;
  h: number;
  l: number;
  c: number;
  v?: number;
}

export interface Chart {
  type: ResultType;
  engine: string;
  title: string;
  symbol?: string;
  currency?: string;
  chartKind: "candlestick" | "line";
  range?: string;
  series: Candle[];
  quote?: Quote;
  url?: string;
}

export const CHART_RANGES = ["1d", "5d", "1mo", "6mo", "ytd", "1y", "5y", "max"] as const;
export type ChartRange = (typeof CHART_RANGES)[number];

// Human labels for the range selector (Google-style).
export const RANGE_LABELS: Record<ChartRange, string> = {
  "1d": "1D",
  "5d": "5D",
  "1mo": "1M",
  "6mo": "6M",
  ytd: "YTD",
  "1y": "1Y",
  "5y": "5Y",
  max: "MAX",
};

// fetchChart refetches a single chart for a symbol at a given range.
// fetchReadme lazily loads a repo's README (plain text) for inline preview.
export async function fetchReadme(fullName: string, signal?: AbortSignal): Promise<string> {
  const res = await fetch(`/api/github/readme?repo=${encodeURIComponent(fullName)}`, { signal });
  if (!res.ok) return "";
  const body = (await res.json().catch(() => ({}))) as { readme?: string };
  return body.readme ?? "";
}

export async function fetchChart(
  symbol: string,
  range: ChartRange,
  signal?: AbortSignal,
): Promise<Chart> {
  const res = await fetch(
    `/api/finance?symbol=${encodeURIComponent(symbol)}&range=${range}`,
    { signal },
  );
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `chart fetch failed: ${res.status}`);
  }
  return (await res.json()) as Chart;
}

export interface Timing {
  engine: string;
  total: number;
}

export interface Unresponsive {
  engine: string;
  reason: string;
}

export interface InfoboxURL {
  title: string;
  url: string;
}
export interface InfoboxAttr {
  label: string;
  value: string;
}
export interface Infobox {
  type: ResultType;
  engine: string;
  title: string;
  id?: string;
  content?: string;
  imgSrc?: string;
  urls?: InfoboxURL[];
  attributes?: InfoboxAttr[];
}

export interface GHRepo {
  type: string;
  engine: string;
  fullName: string;
  url: string;
  description?: string;
  language?: string;
  stars: number;
  forks: number;
  openIssues?: number;
  license?: string;
  updated?: string;
  topics?: string[];
  ownerAvatar?: string;
  archived?: boolean;
}
export interface GHCode {
  type: string;
  engine: string;
  path: string;
  repo: string;
  url: string;
  language?: string;
  fragments?: string[];
}
export interface GHIssue {
  type: string;
  engine: string;
  title: string;
  url: string;
  repo: string;
  number: number;
  state: string;
  isPR: boolean;
  draft?: boolean;
  author?: string;
  authorAvatar?: string;
  comments: number;
  labels?: string[];
  created?: string;
  body?: string;
}
export interface GHUser {
  type: string;
  engine: string;
  login: string;
  url: string;
  avatar?: string;
  name?: string;
  bio?: string;
  isOrg?: boolean;
}
export interface GHTopic {
  type: string;
  engine: string;
  name: string;
  url: string;
  description?: string;
}
export interface GHCommit {
  type: string;
  engine: string;
  sha: string;
  url: string;
  repo: string;
  message: string;
  author?: string;
  date?: string;
}
export interface GHDiscussion {
  type: string;
  engine: string;
  title: string;
  url: string;
  repo: string;
  number: number;
  category?: string;
  author?: string;
  answered?: boolean;
  comments: number;
  created?: string;
  body?: string;
}

export interface GHRankedItem {
  kind: "repo" | "code" | "issue" | "user" | "topic" | "commit" | "discussion";
  score: number;
  repo?: GHRepo;
  code?: GHCode;
  issue?: GHIssue;
  user?: GHUser;
  topic?: GHTopic;
  commit?: GHCommit;
  discussion?: GHDiscussion;
}

export interface VideoResult {
  type: string;
  engine: string;
  title: string;
  url: string;
  content?: string;
  thumbnail?: string;
  author?: string;
  length?: string;
  publishedDate?: string;
}
export interface Paper {
  type: string;
  engine: string;
  title: string;
  url: string;
  content?: string;
  authors?: string[];
  journal?: string;
  publishedDate?: string;
  doi?: string;
  pdfUrl?: string;
  publisher?: string;
  tags?: string[];
}
export interface Torrent {
  type: string;
  engine: string;
  title: string;
  url: string;
  magnetLink?: string;
  torrentFile?: string;
  seeders: number;
  leechers: number;
  fileSize?: string;
  files?: number;
  content?: string;
}
export interface MapResult {
  type: string;
  engine: string;
  title: string;
  url: string;
  content?: string;
  latitude?: number;
  longitude?: number;
  address?: string;
}
export interface CodeResult {
  type: string;
  engine: string;
  title: string;
  url: string;
  content?: string;
  codeSnippet?: string;
  language?: string;
  repository?: string;
  filename?: string;
}
export interface FileResult {
  type: string;
  engine: string;
  title: string;
  url: string;
  content?: string;
  fileSize?: string;
  fileType?: string;
}
export interface KeyValueResult {
  type: string;
  engine: string;
  title?: string;
  url?: string;
  pairs: Record<string, string>;
}

export interface SearchResponse {
  query: string;
  pageno: number;
  results?: MainResult[] | null;
  answers?: Answer[];
  infoboxes?: Infobox[];
  images?: ImageResult[];
  videos?: VideoResult[];
  papers?: Paper[];
  torrents?: Torrent[];
  maps?: MapResult[];
  codes?: CodeResult[];
  files?: FileResult[];
  keyvalues?: KeyValueResult[];
  quotes?: Quote[];
  charts?: Chart[];
  ghRepos?: GHRepo[];
  ghCode?: GHCode[];
  ghIssues?: GHIssue[];
  ghUsers?: GHUser[];
  ghTopics?: GHTopic[];
  ghCommits?: GHCommit[];
  ghDiscussions?: GHDiscussion[];
  ghRanked?: GHRankedItem[];
  suggestions?: string[];
  corrections?: string[];
  timings?: Timing[];
  unresponsive?: Unresponsive[];
}

export interface SearchParams {
  q: string;
  pageno?: number;
  categories?: string[];
  timeRange?: string;
  language?: string;
  safesearch?: number;
}

export async function search(p: SearchParams, signal?: AbortSignal): Promise<SearchResponse> {
  const params = new URLSearchParams({ q: p.q });
  if (p.pageno && p.pageno > 1) params.set("pageno", String(p.pageno));
  if (p.categories && p.categories.length) params.set("categories", p.categories.join(","));
  if (p.timeRange) params.set("time_range", p.timeRange);
  if (p.language) params.set("locale", p.language);
  if (p.safesearch != null) params.set("safesearch", String(p.safesearch));
  const res = await fetch(`/api/search?${params}`, { signal });
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(body.error || `search failed: ${res.status}`);
  }
  return (await res.json()) as SearchResponse;
}

// streamSearch consumes the SSE result stream, invoking onChunk for each
// engine's partial results as they arrive, onDone when complete. Returns an
// abort function. Each chunk is a partial SearchResponse (subset of arrays).
export function streamSearch(
  p: SearchParams,
  onChunk: (chunk: Partial<SearchResponse> & { engine?: string; elapsed?: number }) => void,
  onUnresponsive: (u: Unresponsive) => void,
  onDone: () => void,
  onError: (e: string) => void,
): () => void {
  const params = new URLSearchParams({ q: p.q });
  if (p.pageno && p.pageno > 1) params.set("pageno", String(p.pageno));
  if (p.categories && p.categories.length) params.set("categories", p.categories.join(","));
  if (p.timeRange) params.set("time_range", p.timeRange);
  if (p.language) params.set("locale", p.language);
  if (p.safesearch != null) params.set("safesearch", String(p.safesearch));

  const es = new EventSource(`/api/search/stream?${params}`);
  // Each snapshot is the full merged+ranked state so far; the client REPLACES
  // its state with it (server does the cross-engine dedup/scoring).
  es.addEventListener("snapshot", (e) => {
    try {
      onChunk(JSON.parse((e as MessageEvent).data));
    } catch {
      /* ignore malformed */
    }
  });
  es.addEventListener("chart", (e) => {
    try {
      onChunk({ charts: [JSON.parse((e as MessageEvent).data)] });
    } catch {
      /* ignore */
    }
  });
  es.addEventListener("unresponsive", (e) => {
    try {
      onUnresponsive(JSON.parse((e as MessageEvent).data));
    } catch {
      /* ignore */
    }
  });
  es.addEventListener("done", () => {
    es.close();
    onDone();
  });
  es.onerror = () => {
    es.close();
    onError("stream connection lost");
    onDone();
  };
  return () => es.close();
}

export interface InstanceConfig {
  instance_name: string;
  categories: string[];
  default_category: string;
  autocomplete: string;
  autocomplete_min: number;
  favicon: boolean;
  safe_search: number;
  formats: string[];
  finance: boolean;
  ai: boolean;
  ai_auto: boolean;
  engines_by_category: Record<string, string[]>;
}

export interface AICitation {
  title: string;
  url: string;
}

// askAI streams an LLM-synthesized, cited answer for a query. Returns an abort
// function. onSources fires once with the citations, onDelta per token.
export function askAI(
  q: string,
  categories: string[],
  onSources: (cites: AICitation[]) => void,
  onDelta: (text: string) => void,
  onDone: (answer: string) => void,
  onError: (e: string) => void,
): () => void {
  const params = new URLSearchParams({ q });
  if (categories.length) params.set("categories", categories.join(","));
  const es = new EventSource(`/api/answer?${params}`);
  es.addEventListener("sources", (e) => {
    try {
      onSources(JSON.parse((e as MessageEvent).data));
    } catch {
      /* ignore */
    }
  });
  es.addEventListener("delta", (e) => {
    try {
      onDelta(JSON.parse((e as MessageEvent).data).text);
    } catch {
      /* ignore */
    }
  });
  es.addEventListener("done", (e) => {
    es.close();
    try {
      onDone(JSON.parse((e as MessageEvent).data).answer);
    } catch {
      onDone("");
    }
  });
  es.addEventListener("error", (e) => {
    let msg = "AI request failed";
    try {
      msg = JSON.parse((e as MessageEvent).data).error || msg;
    } catch {
      /* network-level error event has no data */
    }
    es.close();
    onError(msg);
  });
  es.onerror = () => {
    es.close();
    onError("AI stream connection lost");
  };
  return () => es.close();
}

export async function getConfig(): Promise<InstanceConfig> {
  const res = await fetch("/api/config");
  if (!res.ok) throw new Error(`config failed: ${res.status}`);
  return (await res.json()) as InstanceConfig;
}

export interface Preferences {
  language?: string;
  safesearch?: number;
  category?: string;
  autocomplete?: boolean;
  results_new_tab?: boolean;
  disabled_engines?: string[];
}

export async function getPreferences(): Promise<Preferences> {
  const res = await fetch("/api/preferences");
  if (!res.ok) return {};
  return (await res.json()) as Preferences;
}

export async function savePreferences(p: Preferences): Promise<void> {
  await fetch("/api/preferences", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(p),
  });
}

export interface EngineInfo {
  name: string;
  shortcut?: string;
  categories: string[];
  disabled: boolean;
}

export async function getEngines(): Promise<EngineInfo[]> {
  const res = await fetch("/api/engines");
  if (!res.ok) return [];
  const data = (await res.json()) as { engines: EngineInfo[] };
  return data.engines ?? [];
}

export async function autocomplete(q: string, signal?: AbortSignal): Promise<string[]> {
  try {
    const res = await fetch(`/api/autocomplete?q=${encodeURIComponent(q)}`, { signal });
    if (!res.ok) return [];
    const data = (await res.json()) as [string, string[]];
    return Array.isArray(data) && Array.isArray(data[1]) ? data[1] : [];
  } catch {
    return [];
  }
}
