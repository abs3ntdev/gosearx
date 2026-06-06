// templates.go defines the richer result display types ported from SearXNG's
// result_templates/*.html: paper (academic), torrent, map, code, file, keyvalue.
// Each maps to a dedicated React card via the frontend registry.
package result

import "github.com/searxng/gosearx/internal/htmlx"

const (
	TypePaper    Type = "paper"
	TypeTorrent  Type = "torrent"
	TypeMap      Type = "map"
	TypeCode     Type = "code"
	TypeFile     Type = "file"
	TypeKeyValue Type = "keyvalue"
)

// Paper is an academic/scientific publication (arxiv, pubmed, semantic scholar…).
type Paper struct {
	Type          Type     `json:"type"`
	EngineName    string   `json:"engine"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Content       string   `json:"content,omitempty"`
	Authors       []string `json:"authors,omitempty"`
	Journal       string   `json:"journal,omitempty"`
	PublishedDate string   `json:"publishedDate,omitempty"`
	DOI           string   `json:"doi,omitempty"`
	PDFURL        string   `json:"pdfUrl,omitempty"`
	Publisher     string   `json:"publisher,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

func (p *Paper) Kind() Type     { return TypePaper }
func (p *Paper) Engine() string { return p.EngineName }

// Torrent is a torrent search result.
type Torrent struct {
	Type        Type   `json:"type"`
	EngineName  string `json:"engine"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	MagnetLink  string `json:"magnetLink,omitempty"`
	TorrentFile string `json:"torrentFile,omitempty"`
	Seeders     int    `json:"seeders"`
	Leechers    int    `json:"leechers"`
	FileSize    string `json:"fileSize,omitempty"`
	Files       int    `json:"files,omitempty"`
	Content     string `json:"content,omitempty"`
}

func (t *Torrent) Kind() Type     { return TypeTorrent }
func (t *Torrent) Engine() string { return t.EngineName }

// MapResult is a geographic place result.
type MapResult struct {
	Type       Type    `json:"type"`
	EngineName string  `json:"engine"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
	Address    string  `json:"address,omitempty"`
}

func (m *MapResult) Kind() Type     { return TypeMap }
func (m *MapResult) Engine() string { return m.EngineName }

// Code is a generic source-code result (file path + snippet).
type Code struct {
	Type        Type   `json:"type"`
	EngineName  string `json:"engine"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Content     string `json:"content,omitempty"`
	CodeSnippet string `json:"codeSnippet,omitempty"`
	Language    string `json:"language,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Filename    string `json:"filename,omitempty"`
}

func (c *Code) Kind() Type     { return TypeCode }
func (c *Code) Engine() string { return c.EngineName }

// File is a downloadable-file result (with size/type metadata).
type File struct {
	Type       Type   `json:"type"`
	EngineName string `json:"engine"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Content    string `json:"content,omitempty"`
	FileSize   string `json:"fileSize,omitempty"`
	FileType   string `json:"fileType,omitempty"`
}

func (f *File) Kind() Type     { return TypeFile }
func (f *File) Engine() string { return f.EngineName }

// KeyValue is a generic table of attributes (e.g. DB/elasticsearch rows).
type KeyValue struct {
	Type       Type              `json:"type"`
	EngineName string            `json:"engine"`
	Title      string            `json:"title,omitempty"`
	URL        string            `json:"url,omitempty"`
	Pairs      map[string]string `json:"pairs"`
}

func (k *KeyValue) Kind() Type     { return TypeKeyValue }
func (k *KeyValue) Engine() string { return k.EngineName }

// --- FromMap constructors ---

func paperFromMap(engine string, m map[string]any) Result {
	p := &Paper{
		Type: TypePaper, EngineName: engine,
		Title: htmlx.StripHTML(str(m, "title")), URL: str(m, "url"),
		Content: htmlx.SanitizeHTML(str(m, "content")),
		Authors: strSlice(m, "authors"), Journal: str(m, "journal"),
		PublishedDate: str(m, "publishedDate"), DOI: str(m, "doi"),
		PDFURL: str(m, "pdfUrl"), Publisher: str(m, "publisher"),
		Tags: strSlice(m, "tags"),
	}
	if p.URL == "" && p.Title == "" {
		return nil
	}
	return p
}

func torrentFromMap(engine string, m map[string]any) Result {
	t := &Torrent{
		Type: TypeTorrent, EngineName: engine,
		Title: htmlx.StripHTML(str(m, "title")), URL: str(m, "url"),
		MagnetLink: str(m, "magnetLink"), TorrentFile: str(m, "torrentFile"),
		Seeders: intv(m, "seeders"), Leechers: intv(m, "leechers"),
		FileSize: str(m, "fileSize"), Files: intv(m, "files"),
		Content: str(m, "content"),
	}
	if t.URL == "" && t.MagnetLink == "" && t.Title == "" {
		return nil
	}
	return t
}

func mapFromMap(engine string, m map[string]any) Result {
	r := &MapResult{
		Type: TypeMap, EngineName: engine,
		Title: htmlx.StripHTML(str(m, "title")), URL: str(m, "url"),
		Content:  htmlx.SanitizeHTML(str(m, "content")),
		Latitude: num(m, "latitude"), Longitude: num(m, "longitude"),
		Address: str(m, "address"),
	}
	if r.Title == "" {
		return nil
	}
	return r
}

func codeFromMap(engine string, m map[string]any) Result {
	c := &Code{
		Type: TypeCode, EngineName: engine,
		Title: htmlx.StripHTML(str(m, "title")), URL: str(m, "url"),
		Content:     htmlx.SanitizeHTML(str(m, "content")),
		CodeSnippet: str(m, "codeSnippet"), Language: str(m, "language"),
		Repository: str(m, "repository"), Filename: str(m, "filename"),
	}
	if c.URL == "" && c.Title == "" {
		return nil
	}
	return c
}

func fileFromMap(engine string, m map[string]any) Result {
	f := &File{
		Type: TypeFile, EngineName: engine,
		Title: htmlx.StripHTML(str(m, "title")), URL: str(m, "url"),
		Content:  htmlx.SanitizeHTML(str(m, "content")),
		FileSize: str(m, "fileSize"), FileType: str(m, "fileType"),
	}
	if f.URL == "" && f.Title == "" {
		return nil
	}
	return f
}

func keyValueFromMap(engine string, m map[string]any) Result {
	kv := &KeyValue{
		Type: TypeKeyValue, EngineName: engine,
		Title: str(m, "title"), URL: str(m, "url"), Pairs: map[string]string{},
	}
	if p, ok := m["pairs"].(map[string]any); ok {
		for k, v := range p {
			if s, ok := v.(string); ok {
				kv.Pairs[k] = s
			}
		}
	}
	if len(kv.Pairs) == 0 && kv.Title == "" {
		return nil
	}
	return kv
}
