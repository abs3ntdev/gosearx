// frommap.go builds typed results from generic maps. Used by the plugin and
// script tiers (and any future tier) to convert dynamic Lua/JSON output into the
// typed result union without each tier re-implementing the dispatch.
package result

import "github.com/searxng/gosearx/internal/htmlx"

// FromMap converts a generic result map into a typed Result. The "type" key (or
// presence of discriminating keys like "suggestion"/"answer") selects the type.
// Returns nil if the map has no usable content.
func FromMap(engine string, m map[string]any) Result {
	if s := str(m, "suggestion"); s != "" {
		return &Suggestion{EngineName: engine, Value: s}
	}
	if a := str(m, "answer"); a != "" {
		return &Answer{EngineName: engine, Answer: a, URL: str(m, "url")}
	}

	if c := str(m, "correction"); c != "" {
		return &Correction{EngineName: engine, Value: c}
	}

	switch Type(str(m, "type")) {
	case TypeAnswer:
		return &Answer{EngineName: engine, Answer: str(m, "answer"), URL: str(m, "url")}
	case TypeCorrection:
		return &Correction{EngineName: engine, Value: str(m, "correction")}
	case TypeInfobox:
		return infoboxFromMap(engine, m)
	case TypeImage:
		return imageFromMap(engine, m)
	case TypeVideo:
		return videoFromMap(engine, m)
	case TypePaper:
		return paperFromMap(engine, m)
	case TypeTorrent:
		return torrentFromMap(engine, m)
	case TypeMap:
		return mapFromMap(engine, m)
	case TypeCode:
		return codeFromMap(engine, m)
	case TypeFile:
		return fileFromMap(engine, m)
	case TypeKeyValue:
		return keyValueFromMap(engine, m)
	case TypeQuote:
		return quoteFromMap(engine, m)
	case TypeChart:
		return chartFromMap(engine, m)
	case TypeGHRepo:
		return ghRepoFromMap(engine, m)
	case TypeGHCode:
		return ghCodeFromMap(engine, m)
	case TypeGHIssue:
		return ghIssueFromMap(engine, m)
	case TypeGHUser:
		return ghUserFromMap(engine, m)
	case TypeGHTopic:
		return &GHTopic{Type: TypeGHTopic, EngineName: engine, Name: str(m, "name"),
			URL: str(m, "url"), Description: htmlx.StripHTML(str(m, "description"))}
	case TypeGHCommit:
		return ghCommitFromMap(engine, m)
	case TypeGHDiscussion:
		return ghDiscussionFromMap(engine, m)
	}

	mr := &MainResult{
		Type:          TypeMain,
		EngineName:    engine,
		Title:         htmlx.StripHTML(str(m, "title")), // titles are plain text
		URL:           str(m, "url"),
		Content:       htmlx.SanitizeHTML(str(m, "content")), // content keeps safe formatting
		Thumbnail:     str(m, "thumbnail"),
		PublishedDate: str(m, "publishedDate"),
	}
	if mr.URL == "" && mr.Title == "" {
		return nil
	}
	return mr
}

func str(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func num(m map[string]any, k string) float64 {
	if v, ok := m[k]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}
