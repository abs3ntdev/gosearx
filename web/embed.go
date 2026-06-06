// Package web embeds the built React frontend (web/dist) into the binary so
// gosearx ships as a single self-contained executable. Run `npm run build` in
// this directory before `go build` to refresh the embedded assets.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Assets returns the embedded frontend filesystem rooted at dist/, or false if
// the frontend has not been built yet (dist/ empty/missing).
func Assets() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
