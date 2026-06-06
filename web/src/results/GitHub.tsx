// Rich GitHub result cards — repos, code, issues/PRs, users, topics.
// These are the first-class GitHub display types, far beyond a plain web link.
import { useState, useCallback } from "react";
import type {
  GHRepo,
  GHCode,
  GHIssue,
  GHUser,
  GHTopic,
  GHCommit,
  GHDiscussion,
  GHRankedItem,
} from "../api";
import { fetchReadme } from "../api";

// arr coerces a possibly-null/undefined value into a safe array.
function arr<T>(v: T[] | null | undefined): T[] {
  return Array.isArray(v) ? v : [];
}

// CopyButton copies the given text to the clipboard and flips its label briefly.
function CopyButton({ text, label = "Copy" }: { text: string; label?: string }): React.JSX.Element {
  const [copied, setCopied] = useState(false);
  const copy = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      navigator.clipboard.writeText(text).then(
        () => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1200);
        },
        () => {},
      );
    },
    [text],
  );
  return (
    <button
      type="button"
      className={"gh-copy" + (copied ? " copied" : "")}
      onClick={copy}
      title="Copy to clipboard"
    >
      {copied ? "✓ Copied" : label}
    </button>
  );
}

function fmtNum(n: number | null | undefined): string {
  const x = n ?? 0;
  if (x >= 1000) return (x / 1000).toFixed(x >= 10000 ? 0 : 1) + "k";
  return String(x);
}

function fmtDate(s?: string): string {
  if (!s) return "";
  const d = new Date(s);
  if (isNaN(+d)) return "";
  return d.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

export function RepoCard({ repo }: { repo: GHRepo }): React.JSX.Element {
  const [readme, setReadme] = useState<string | null>(null);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const toggleReadme = useCallback(() => {
    if (open) {
      setOpen(false);
      return;
    }
    setOpen(true);
    if (readme === null && !loading) {
      setLoading(true);
      fetchReadme(repo.fullName)
        .then((md) => setReadme(md || ""))
        .catch(() => setReadme(""))
        .finally(() => setLoading(false));
    }
  }, [open, readme, loading, repo.fullName]);

  return (
    <article className="gh-card gh-repo">
      {repo.ownerAvatar && <img className="gh-avatar" src={repo.ownerAvatar} alt="" loading="lazy" />}
      <div className="gh-body">
        <h3 className="gh-title">
          <a href={repo.url} target="_blank" rel="noreferrer">
            {repo.fullName}
          </a>
          {repo.archived && <span className="gh-archived">archived</span>}
          <button type="button" className="gh-readme-toggle" onClick={toggleReadme}>
            {open ? "Hide README" : "README"}
          </button>
        </h3>
        {repo.description && <p className="gh-desc">{repo.description}</p>}
        {arr(repo.topics).length > 0 && (
          <div className="gh-topics">
            {arr(repo.topics)
              .slice(0, 6)
              .map((t) => (
                <span className="gh-topic" key={t}>
                  {t}
                </span>
              ))}
          </div>
        )}
        <div className="gh-meta">
          {repo.language && (
            <span className="gh-lang">
              <span className="gh-lang-dot" /> {repo.language}
            </span>
          )}
          <span title="stars">★ {fmtNum(repo.stars)}</span>
          <span title="forks">⑂ {fmtNum(repo.forks)}</span>
          {repo.openIssues != null && <span title="open issues">⊙ {fmtNum(repo.openIssues)}</span>}
          {repo.license && <span className="gh-license">{repo.license}</span>}
          {repo.updated && <span className="gh-updated">updated {fmtDate(repo.updated)}</span>}
        </div>
        {open && (
          <div className="gh-readme">
            {loading && <span className="gh-readme-loading">Loading README…</span>}
            {!loading && readme === "" && <span className="gh-readme-loading">No README found.</span>}
            {!loading && readme && <pre className="gh-readme-body">{readme}</pre>}
          </div>
        )}
      </div>
    </article>
  );
}

export function CodeCard({ code }: { code: GHCode }): React.JSX.Element {
  const snippet = arr(code.fragments).join("\n…\n");
  return (
    <article className="gh-card gh-code">
      <div className="gh-code-head">
        <a href={code.url} target="_blank" rel="noreferrer">
          <span className="gh-repo-name">{code.repo}</span>
          <span className="gh-path"> / {code.path}</span>
        </a>
      </div>
      {snippet.length > 0 && (
        <pre className="gh-snippet">
          <CopyButton text={snippet} />
          <code>{snippet}</code>
        </pre>
      )}
    </article>
  );
}

export function IssueCard({ issue }: { issue: GHIssue }): React.JSX.Element {
  const stateClass =
    issue.state === "merged"
      ? "gh-state merged"
      : issue.state === "closed"
        ? "gh-state closed"
        : "gh-state open";
  return (
    <article className="gh-card gh-issue">
      <div className="gh-issue-head">
        <span className={stateClass}>
          {issue.isPR ? "PR" : "Issue"} · {issue.draft ? "draft" : issue.state}
        </span>
        <a href={issue.url} target="_blank" rel="noreferrer" className="gh-issue-title">
          {issue.title}
        </a>
      </div>
      <div className="gh-meta">
        <span className="gh-repo-name">{issue.repo}</span>
        <span>#{issue.number}</span>
        {issue.author && <span>by {issue.author}</span>}
        {issue.comments > 0 && <span>💬 {issue.comments}</span>}
        {issue.created && <span>{fmtDate(issue.created)}</span>}
      </div>
      {arr(issue.labels).length > 0 && (
        <div className="gh-labels">
          {arr(issue.labels)
            .slice(0, 6)
            .map((l) => (
              <span className="gh-label" key={l}>
                {l}
              </span>
            ))}
        </div>
      )}
    </article>
  );
}

export function UserCard({ user }: { user: GHUser }): React.JSX.Element {
  return (
    <a className="gh-card gh-user" href={user.url} target="_blank" rel="noreferrer">
      {user.avatar && <img className="gh-avatar" src={user.avatar} alt="" loading="lazy" />}
      <div className="gh-body">
        <span className="gh-login">{user.login}</span>
        <span className="gh-user-type">{user.isOrg ? "Organization" : "User"}</span>
      </div>
    </a>
  );
}

export function CommitCard({ commit }: { commit: GHCommit }): React.JSX.Element {
  return (
    <article className="gh-card gh-commit">
      <div className="gh-body">
        <a className="gh-commit-msg" href={commit.url} target="_blank" rel="noreferrer">
          {commit.message}
        </a>
        <div className="gh-meta">
          <span className="gh-repo-name">{commit.repo}</span>
          <code className="gh-sha">{commit.sha}</code>
          <CopyButton text={commit.sha} label="⧉" />
          {commit.author && <span>{commit.author}</span>}
          {commit.date && <span>{fmtDate(commit.date)}</span>}
        </div>
      </div>
    </article>
  );
}

export function DiscussionCard({ d }: { d: GHDiscussion }): React.JSX.Element {
  return (
    <article className="gh-card gh-discussion">
      <div className="gh-issue-head">
        <span className={d.answered ? "gh-state merged" : "gh-state open"}>
          {d.answered ? "answered" : "discussion"}
        </span>
        <a href={d.url} target="_blank" rel="noreferrer" className="gh-issue-title">
          {d.title}
        </a>
      </div>
      <div className="gh-meta">
        <span className="gh-repo-name">{d.repo}</span>
        {d.category && <span className="gh-topic">{d.category}</span>}
        {d.author && <span>by {d.author}</span>}
        {d.comments > 0 && <span>💬 {d.comments}</span>}
        {d.created && <span>{fmtDate(d.created)}</span>}
      </div>
    </article>
  );
}

export function TopicCard({ topic }: { topic: GHTopic }): React.JSX.Element {
  return (
    <a className="gh-card gh-topic-card" href={topic.url} target="_blank" rel="noreferrer">
      <span className="gh-topic-name"># {topic.name}</span>
      {topic.description && <span className="gh-desc">{topic.description}</span>}
    </a>
  );
}

// renderRanked dispatches one ranked item to its type-specific card.
function renderRanked(item: GHRankedItem, key: number): React.JSX.Element | null {
  switch (item.kind) {
    case "repo":
      return item.repo ? <RepoCard repo={item.repo} key={key} /> : null;
    case "issue":
      return item.issue ? <IssueCard issue={item.issue} key={key} /> : null;
    case "discussion":
      return item.discussion ? <DiscussionCard d={item.discussion} key={key} /> : null;
    case "code":
      return item.code ? <CodeCard code={item.code} key={key} /> : null;
    case "user":
      return item.user ? <UserCard user={item.user} key={key} /> : null;
    case "commit":
      return item.commit ? <CommitCard commit={item.commit} key={key} /> : null;
    case "topic":
      return item.topic ? <TopicCard topic={item.topic} key={key} /> : null;
    default:
      return null;
  }
}

// GitHubResults renders the unified relevance-ranked GitHub list (best match
// first, mixed types) when available, falling back to per-type sections.
export function GitHubResults({ resp }: { resp: import("../api").SearchResponse }): React.JSX.Element | null {
  const ranked = arr(resp.ghRanked);
  if (ranked.length > 0) {
    return (
      <div className="gh-results gh-ranked">
        {ranked.map((it, i) => (
          <div className="gh-ranked-item" key={i}>
            <span className={"gh-kind gh-kind-" + it.kind}>{it.kind}</span>
            {renderRanked(it, i)}
          </div>
        ))}
      </div>
    );
  }

  const repos = arr(resp.ghRepos);
  const code = arr(resp.ghCode);
  const issues = arr(resp.ghIssues);
  const users = arr(resp.ghUsers);
  const topics = arr(resp.ghTopics);
  const commits = arr(resp.ghCommits);
  const discussions = arr(resp.ghDiscussions);
  if (
    !repos.length &&
    !code.length &&
    !issues.length &&
    !users.length &&
    !topics.length &&
    !commits.length &&
    !discussions.length
  ) {
    return null;
  }
  return (
    <div className="gh-results">
      {repos.length > 0 && (
        <section>
          <h2 className="gh-section">Repositories</h2>
          {repos.map((r, i) => (
            <RepoCard repo={r} key={i} />
          ))}
        </section>
      )}
      {issues.length > 0 && (
        <section>
          <h2 className="gh-section">Issues &amp; Pull Requests</h2>
          {issues.map((r, i) => (
            <IssueCard issue={r} key={i} />
          ))}
        </section>
      )}
      {discussions.length > 0 && (
        <section>
          <h2 className="gh-section">Discussions</h2>
          {discussions.map((r, i) => (
            <DiscussionCard d={r} key={i} />
          ))}
        </section>
      )}
      {code.length > 0 && (
        <section>
          <h2 className="gh-section">Code</h2>
          {code.map((r, i) => (
            <CodeCard code={r} key={i} />
          ))}
        </section>
      )}
      {commits.length > 0 && (
        <section>
          <h2 className="gh-section">Commits</h2>
          {commits.map((r, i) => (
            <CommitCard commit={r} key={i} />
          ))}
        </section>
      )}
      {users.length > 0 && (
        <section>
          <h2 className="gh-section">Users &amp; Organizations</h2>
          <div className="gh-user-grid">
            {users.map((r, i) => (
              <UserCard user={r} key={i} />
            ))}
          </div>
        </section>
      )}
      {topics.length > 0 && (
        <section>
          <h2 className="gh-section">Topics</h2>
          <div className="gh-topic-grid">
            {topics.map((r, i) => (
              <TopicCard topic={r} key={i} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
