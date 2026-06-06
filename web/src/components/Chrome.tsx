// Chrome.tsx holds the UI "chrome" around results: category tabs, the filter
// bar, answers, corrections, the sidebar (infoboxes/suggestions/APIs), and
// pagination. Mirrors SearXNG's simple theme layout.
import type { Answer, Infobox } from "../api";

const CATEGORY_ICONS: Record<string, string> = {
  general: "🔍",
  images: "🖼️",
  videos: "▶️",
  news: "📰",
  map: "📍",
  music: "🎵",
  it: "💻",
  github: "🐙",
  science: "🎓",
  files: "📁",
  "social media": "👥",
};

export function CategoryTabs({
  categories,
  active,
  onSelect,
}: {
  categories: string[];
  active: string;
  onSelect: (c: string) => void;
}): React.JSX.Element {
  return (
    <nav className="category-tabs">
      {categories.map((c) => (
        <button
          key={c}
          className={c === active ? "cat-tab active" : "cat-tab"}
          onClick={() => onSelect(c)}
        >
          <span className="cat-icon">{CATEGORY_ICONS[c] ?? "•"}</span>
          {c}
        </button>
      ))}
    </nav>
  );
}

const TIME_RANGES = [
  { v: "", label: "Anytime" },
  { v: "day", label: "Day" },
  { v: "week", label: "Week" },
  { v: "month", label: "Month" },
  { v: "year", label: "Year" },
];

const LANGUAGES = [
  { v: "all", label: "Any language" },
  { v: "en", label: "English" },
  { v: "en-US", label: "English (US)" },
  { v: "de", label: "Deutsch" },
  { v: "fr", label: "Français" },
  { v: "es", label: "Español" },
  { v: "it", label: "Italiano" },
  { v: "ja", label: "日本語" },
  { v: "zh", label: "中文" },
];

const FILETYPES = [
  { v: "", label: "Any type" },
  { v: "pdf", label: "PDF" },
  { v: "doc", label: "Word" },
  { v: "ppt", label: "PowerPoint" },
  { v: "xls", label: "Excel" },
  { v: "txt", label: "Text" },
  { v: "csv", label: "CSV" },
];

export interface SearchFilters {
  fileType: string;
  siteInclude: string;
  siteExclude: string;
}

export function FilterBar({
  timeRange,
  language,
  safesearch,
  filters,
  onTimeRange,
  onLanguage,
  onSafesearch,
  onFilters,
}: {
  timeRange: string;
  language: string;
  safesearch: number;
  filters: SearchFilters;
  onTimeRange: (v: string) => void;
  onLanguage: (v: string) => void;
  onSafesearch: (v: number) => void;
  onFilters: (f: SearchFilters) => void;
}): React.JSX.Element {
  return (
    <div className="filter-bar">
      <label>
        Time
        <select value={timeRange} onChange={(e) => onTimeRange(e.target.value)}>
          {TIME_RANGES.map((t) => (
            <option key={t.v} value={t.v}>
              {t.label}
            </option>
          ))}
        </select>
      </label>
      <label>
        Language
        <select value={language} onChange={(e) => onLanguage(e.target.value)}>
          {LANGUAGES.map((l) => (
            <option key={l.v} value={l.v}>
              {l.label}
            </option>
          ))}
        </select>
      </label>
      <label>
        SafeSearch
        <select value={safesearch} onChange={(e) => onSafesearch(Number(e.target.value))}>
          <option value={0}>None</option>
          <option value={1}>Moderate</option>
          <option value={2}>Strict</option>
        </select>
      </label>
      <label>
        File type
        <select
          value={filters.fileType}
          onChange={(e) => onFilters({ ...filters, fileType: e.target.value })}
        >
          {FILETYPES.map((t) => (
            <option key={t.v} value={t.v}>
              {t.label}
            </option>
          ))}
        </select>
      </label>
      <label className="filter-text">
        Site
        <input
          type="text"
          placeholder="example.com"
          value={filters.siteInclude}
          onChange={(e) => onFilters({ ...filters, siteInclude: e.target.value })}
          onBlur={(e) => onFilters({ ...filters, siteInclude: e.target.value.trim() })}
        />
      </label>
      <label className="filter-text">
        Exclude
        <input
          type="text"
          placeholder="-spam.com"
          value={filters.siteExclude}
          onChange={(e) => onFilters({ ...filters, siteExclude: e.target.value })}
          onBlur={(e) => onFilters({ ...filters, siteExclude: e.target.value.trim() })}
        />
      </label>
    </div>
  );
}

export function Answers({ answers }: { answers: Answer[] }): React.JSX.Element {
  return (
    <div className="answers">
      {answers.map((a, i) => (
        <div className="answer-card" key={i}>
          <span className="answer-text">{a.answer}</span>
          {a.url && (
            <a className="answer-link" href={a.url} target="_blank" rel="noreferrer">
              source
            </a>
          )}
        </div>
      ))}
    </div>
  );
}

export function Corrections({
  corrections,
  onPick,
}: {
  corrections: string[];
  onPick: (c: string) => void;
}): React.JSX.Element {
  return (
    <div className="corrections">
      <span className="corrections-label">Did you mean:</span>
      {corrections.map((c) => (
        <button key={c} className="correction" onClick={() => onPick(c)}>
          {c}
        </button>
      ))}
    </div>
  );
}

export function InfoboxCard({ box }: { box: Infobox }): React.JSX.Element {
  return (
    <div className="infobox">
      {box.imgSrc && <img className="infobox-img" src={box.imgSrc} alt={box.title} />}
      <h3 className="infobox-title">{box.title}</h3>
      {box.content && <p className="infobox-content">{box.content}</p>}
      {box.attributes && box.attributes.length > 0 && (
        <table className="infobox-attrs">
          <tbody>
            {box.attributes.map((a, i) => (
              <tr key={i}>
                <td className="attr-label">{a.label}</td>
                <td className="attr-value">{a.value}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {box.urls && box.urls.length > 0 && (
        <ul className="infobox-links">
          {box.urls.map((u, i) => (
            <li key={i}>
              <a href={u.url} target="_blank" rel="noreferrer">
                {u.title}
              </a>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function Suggestions({
  suggestions,
  onPick,
}: {
  suggestions: string[];
  onPick: (s: string) => void;
}): React.JSX.Element {
  return (
    <div className="sidebar-box">
      <h4 className="sidebar-title">Suggestions</h4>
      <ul className="suggestion-list">
        {suggestions.map((s) => (
          <li key={s}>
            <button className="suggestion" onClick={() => onPick(s)}>
              {s}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function SearchFormats({ query }: { query: string }): React.JSX.Element {
  const q = encodeURIComponent(query);
  return (
    <div className="sidebar-box">
      <h4 className="sidebar-title">Download results</h4>
      <div className="format-links">
        <a href={`/api/search?q=${q}&format=json`} target="_blank" rel="noreferrer">
          JSON
        </a>
        <a href={`/api/search?q=${q}&format=csv`} target="_blank" rel="noreferrer">
          CSV
        </a>
        <a href={`/api/search?q=${q}&format=rss`} target="_blank" rel="noreferrer">
          RSS
        </a>
      </div>
    </div>
  );
}

export function Pagination({
  pageno,
  hasResults,
  onPage,
}: {
  pageno: number;
  hasResults: boolean;
  onPage: (p: number) => void;
}): React.JSX.Element {
  return (
    <div className="pagination">
      <button disabled={pageno <= 1} onClick={() => onPage(pageno - 1)}>
        ‹ Previous
      </button>
      <span className="page-num">Page {pageno}</span>
      <button disabled={!hasResults} onClick={() => onPage(pageno + 1)}>
        Next ›
      </button>
    </div>
  );
}
