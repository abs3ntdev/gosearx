// github.go defines rich, first-class GitHub result types. These go well beyond
// a plain web link: repos carry stars/language/license, issues/PRs carry state +
// labels, code results carry a file path + snippet, users carry avatar + bio.
// Each maps to a dedicated React card in the frontend registry.
package result

import "github.com/searxng/gosearx/internal/htmlx"

const (
	TypeGHRepo       Type = "gh_repo"
	TypeGHCode       Type = "gh_code"
	TypeGHIssue      Type = "gh_issue" // also used for PRs (IsPR distinguishes)
	TypeGHUser       Type = "gh_user"
	TypeGHTopic      Type = "gh_topic"
	TypeGHCommit     Type = "gh_commit"
	TypeGHDiscussion Type = "gh_discussion"
)

// GHRepo is a repository result.
type GHRepo struct {
	Type        Type     `json:"type"`
	EngineName  string   `json:"engine"`
	FullName    string   `json:"fullName"`
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	Language    string   `json:"language,omitempty"`
	Stars       int      `json:"stars"`
	Forks       int      `json:"forks"`
	OpenIssues  int      `json:"openIssues,omitempty"`
	License     string   `json:"license,omitempty"`
	Updated     string   `json:"updated,omitempty"`
	Topics      []string `json:"topics,omitempty"`
	OwnerAvatar string   `json:"ownerAvatar,omitempty"`
	Archived    bool     `json:"archived,omitempty"`
}

func (r *GHRepo) Kind() Type     { return TypeGHRepo }
func (r *GHRepo) Engine() string { return r.EngineName }

// GHCode is a code search result (a matching file).
type GHCode struct {
	Type       Type     `json:"type"`
	EngineName string   `json:"engine"`
	Path       string   `json:"path"`
	Repo       string   `json:"repo"`
	URL        string   `json:"url"`
	Language   string   `json:"language,omitempty"`
	Fragments  []string `json:"fragments,omitempty"` // matched code lines
}

func (c *GHCode) Kind() Type     { return TypeGHCode }
func (c *GHCode) Engine() string { return c.EngineName }

// GHIssue is an issue or pull request.
type GHIssue struct {
	Type         Type     `json:"type"`
	EngineName   string   `json:"engine"`
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	Repo         string   `json:"repo"`
	Number       int      `json:"number"`
	State        string   `json:"state"` // open | closed | merged
	IsPR         bool     `json:"isPR"`
	Draft        bool     `json:"draft,omitempty"`
	Author       string   `json:"author,omitempty"`
	AuthorAvatar string   `json:"authorAvatar,omitempty"`
	Comments     int      `json:"comments"`
	Labels       []string `json:"labels,omitempty"`
	Created      string   `json:"created,omitempty"`
	Body         string   `json:"body,omitempty"`
}

func (i *GHIssue) Kind() Type     { return TypeGHIssue }
func (i *GHIssue) Engine() string { return i.EngineName }

// GHUser is a user or organization.
type GHUser struct {
	Type       Type   `json:"type"`
	EngineName string `json:"engine"`
	Login      string `json:"login"`
	URL        string `json:"url"`
	Avatar     string `json:"avatar,omitempty"`
	Name       string `json:"name,omitempty"`
	Bio        string `json:"bio,omitempty"`
	IsOrg      bool   `json:"isOrg,omitempty"`
}

func (u *GHUser) Kind() Type     { return TypeGHUser }
func (u *GHUser) Engine() string { return u.EngineName }

// GHTopic is a GitHub topic.
type GHTopic struct {
	Type        Type   `json:"type"`
	EngineName  string `json:"engine"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

func (t *GHTopic) Kind() Type     { return TypeGHTopic }
func (t *GHTopic) Engine() string { return t.EngineName }

// GHCommit is a commit search result.
type GHCommit struct {
	Type       Type   `json:"type"`
	EngineName string `json:"engine"`
	SHA        string `json:"sha"`
	URL        string `json:"url"`
	Repo       string `json:"repo"`
	Message    string `json:"message"`
	Author     string `json:"author,omitempty"`
	Date       string `json:"date,omitempty"`
}

func (c *GHCommit) Kind() Type     { return TypeGHCommit }
func (c *GHCommit) Engine() string { return c.EngineName }

// GHDiscussion is a GitHub Discussion search result.
type GHDiscussion struct {
	Type       Type   `json:"type"`
	EngineName string `json:"engine"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Repo       string `json:"repo"`
	Number     int    `json:"number"`
	Category   string `json:"category,omitempty"`
	Author     string `json:"author,omitempty"`
	Answered   bool   `json:"answered,omitempty"`
	Comments   int    `json:"comments"`
	Created    string `json:"created,omitempty"`
	Body       string `json:"body,omitempty"`
}

func (d *GHDiscussion) Kind() Type     { return TypeGHDiscussion }
func (d *GHDiscussion) Engine() string { return d.EngineName }

// --- FromMap constructors (convert Lua/JSON maps into the typed results) ---

func ghRepoFromMap(engine string, m map[string]any) Result {
	r := &GHRepo{
		Type: TypeGHRepo, EngineName: engine,
		FullName: str(m, "fullName"), URL: str(m, "url"),
		Description: str(m, "description"), Language: str(m, "language"),
		Stars: intv(m, "stars"), Forks: intv(m, "forks"),
		OpenIssues: intv(m, "openIssues"), License: str(m, "license"),
		Updated: str(m, "updated"), OwnerAvatar: str(m, "ownerAvatar"),
		Topics:   strSlice(m, "topics"),
		Archived: boolv(m, "archived"),
	}
	if r.URL == "" && r.FullName == "" {
		return nil
	}
	return r
}

func ghCodeFromMap(engine string, m map[string]any) Result {
	c := &GHCode{
		Type: TypeGHCode, EngineName: engine,
		Path: str(m, "path"), Repo: str(m, "repo"), URL: str(m, "url"),
		Language: str(m, "language"), Fragments: strSlice(m, "fragments"),
	}
	if c.URL == "" {
		return nil
	}
	return c
}

func ghIssueFromMap(engine string, m map[string]any) Result {
	i := &GHIssue{
		Type: TypeGHIssue, EngineName: engine,
		Title: str(m, "title"), URL: str(m, "url"), Repo: str(m, "repo"),
		Number: intv(m, "number"), State: str(m, "state"), IsPR: boolv(m, "isPR"),
		Draft: boolv(m, "draft"), Author: str(m, "author"),
		AuthorAvatar: str(m, "authorAvatar"), Comments: intv(m, "comments"),
		Labels: strSlice(m, "labels"), Created: str(m, "created"),
		Body: htmlx.StripHTML(str(m, "body")),
	}
	if i.URL == "" {
		return nil
	}
	return i
}

func ghCommitFromMap(engine string, m map[string]any) Result {
	c := &GHCommit{
		Type: TypeGHCommit, EngineName: engine,
		SHA: str(m, "sha"), URL: str(m, "url"), Repo: str(m, "repo"),
		Message: str(m, "message"), Author: str(m, "author"), Date: str(m, "date"),
	}
	if c.URL == "" {
		return nil
	}
	return c
}

func ghDiscussionFromMap(engine string, m map[string]any) Result {
	d := &GHDiscussion{
		Type: TypeGHDiscussion, EngineName: engine,
		Title: str(m, "title"), URL: str(m, "url"), Repo: str(m, "repo"),
		Number: intv(m, "number"), Category: str(m, "category"),
		Author: str(m, "author"), Answered: boolv(m, "answered"),
		Comments: intv(m, "comments"), Created: str(m, "created"),
		Body: htmlx.StripHTML(str(m, "body")),
	}
	if d.URL == "" {
		return nil
	}
	return d
}

func ghUserFromMap(engine string, m map[string]any) Result {
	u := &GHUser{
		Type: TypeGHUser, EngineName: engine,
		Login: str(m, "login"), URL: str(m, "url"), Avatar: str(m, "avatar"),
		Name: str(m, "name"), Bio: str(m, "bio"), IsOrg: boolv(m, "isOrg"),
	}
	if u.URL == "" && u.Login == "" {
		return nil
	}
	return u
}

func intv(m map[string]any, k string) int {
	if v, ok := m[k]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return 0
}

func boolv(m map[string]any, k string) bool {
	if v, ok := m[k]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func strSlice(m map[string]any, k string) []string {
	v, ok := m[k]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
