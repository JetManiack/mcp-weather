// Package frontend embeds and serves the admin web interface.
package frontend

import (
	"embed"
	"net/http"
)

//go:embed static
var staticFS embed.FS

// Handler returns an http.Handler that serves the embedded admin UI at the
// mount root. Files are served from the embedded "static" subdirectory, so a
// request for "/" maps to "static/index.html".
func Handler() http.Handler {
	return http.FileServer(http.FS(staticFS))
}
