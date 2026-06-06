// Package github provides intent detection for GitHub searches — the routing
// brain that makes the GitHub integration first-class. Given a raw query it
// decides which GitHub search engines are most relevant and how to rewrite the
// query (adding qualifiers), so "react useState bug", "facebook/react", "@torvalds",
// "is:pr draft", and "#1234 in owner/repo" each do the right thing.
package github

import (
	"regexp"
	"strings"
)

// Intent describes the routing decision for a GitHub query.
type Intent struct {
	// Engines is the ordered set of GitHub engine names to run.
	Engines []string
	// Query is the (possibly rewritten) query to send.
	Query string
}

var (
	reOwnerRepo = regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)
	reUserAt    = regexp.MustCompile(`^@([\w-]+)$`)
	reIssueNum  = regexp.MustCompile(`(^|\s)#(\d+)(\s|$)`)
)

// Detect routes a raw GitHub query to the most relevant engines.
//
// Heuristics (first match wins for the primary engine; extras may be added):
//   - "owner/repo"          -> that repo + its issues/PRs
//   - "@user" / "user:x"    -> users
//   - contains "is:pr"      -> PRs
//   - contains "is:issue" / "#123" / "label:" -> issues
//   - contains "lang:"/"language:"/"path:"/"extension:"/"in:file" -> code
//   - "topic:x" or "#topic" -> topics
//   - otherwise (bare)      -> repos + code + issues (broad)
func Detect(raw string) Intent {
	q := strings.TrimSpace(raw)
	low := strings.ToLower(q)

	switch {
	case reOwnerRepo.MatchString(q):
		// A specific repo: show the repo, plus its open issues and PRs.
		return Intent{
			Engines: []string{"github", "github_issues", "github_prs"},
			Query:   "repo:" + q,
		}

	case reUserAt.MatchString(q):
		m := reUserAt.FindStringSubmatch(q)
		return Intent{Engines: []string{"github_users"}, Query: m[1]}

	case strings.Contains(low, "user:") || strings.Contains(low, "org:") && !strings.Contains(low, "repo:"):
		return Intent{Engines: []string{"github_users", "github"}, Query: q}

	case strings.Contains(low, "is:pr") || strings.HasPrefix(low, "pr ") || strings.Contains(low, "pull request"):
		return Intent{Engines: []string{"github_prs"}, Query: stripWord(q, "pr")}

	case strings.Contains(low, "is:issue") || reIssueNum.MatchString(q) ||
		strings.Contains(low, "label:") || strings.HasPrefix(low, "issue "):
		return Intent{Engines: []string{"github_issues"}, Query: stripWord(q, "issue")}

	case strings.Contains(low, "lang:") || strings.Contains(low, "language:") ||
		strings.Contains(low, "path:") || strings.Contains(low, "extension:") ||
		strings.Contains(low, "filename:") || strings.Contains(low, "in:file"):
		return Intent{Engines: []string{"github_code", "github"}, Query: q}

	case strings.Contains(low, "topic:") || strings.HasPrefix(q, "#"):
		return Intent{Engines: []string{"github_topics"}, Query: strings.TrimPrefix(q, "#")}

	case strings.HasPrefix(low, "commit ") || strings.Contains(low, "committer:") ||
		strings.Contains(low, "author-name:"):
		return Intent{Engines: []string{"github_commits"}, Query: stripWord(q, "commit")}

	case strings.HasPrefix(low, "discussion ") || strings.Contains(low, "is:discussion"):
		return Intent{Engines: []string{"github_discussions"}, Query: stripWord(q, "discussion")}

	default:
		// Broad: repos primary; code + issues + discussions add depth.
		return Intent{
			Engines: []string{"github", "github_code", "github_issues", "github_users", "github_discussions"},
			Query:   q,
		}
	}
}

// stripWord removes a leading command word ("pr"/"issue") from the query.
func stripWord(q, word string) string {
	low := strings.ToLower(q)
	if strings.HasPrefix(low, word+" ") {
		return strings.TrimSpace(q[len(word)+1:])
	}
	return q
}
