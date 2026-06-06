// formats.go implements the CSV and RSS output formats for /api/search,
// mirroring SearXNG's format=csv and format=rss. JSON is the default.
package server

import (
	"encoding/csv"
	"encoding/xml"
	"net/http"
	"strconv"
	"time"

	"github.com/searxng/gosearx/internal/result"
)

// writeCSV emits results as CSV (columns: title, url, content, engine, score).
func writeCSV(w http.ResponseWriter, results []*result.MainResult) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="searx_results.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"title", "url", "content", "engine", "score"})
	for _, r := range results {
		_ = cw.Write([]string{
			r.Title, r.URL, r.Content, r.EngineName,
			strconv.FormatFloat(r.Score, 'f', 4, 64),
		})
	}
	cw.Flush()
}

// rss structures for encoding/xml.
type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

// writeRSS emits results as an RSS 2.0 feed (mirrors SearXNG's opensearch RSS).
func writeRSS(w http.ResponseWriter, query string, results []*result.MainResult) {
	root := rssRoot{
		Version: "2.0",
		Channel: rssChannel{
			Title:       "gosearx: " + query,
			Link:        "",
			Description: "Search results for " + query,
			PubDate:     time.Now().UTC().Format(time.RFC1123Z),
		},
	}
	for _, r := range results {
		root.Channel.Items = append(root.Channel.Items, rssItem{
			Title:       r.Title,
			Link:        r.URL,
			Description: r.Content,
		})
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(root)
}
