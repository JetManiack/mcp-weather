// Package ui embeds and serves the admin web interface.
package ui

import (
	"embed"
	"net/http"
)

//go:embed static
var staticFS embed.FS

// NewHandler returns an http.Handler that serves the embedded admin UI.
func NewHandler() http.Handler {
	return http.FileServer(http.FS(staticFS))
}
