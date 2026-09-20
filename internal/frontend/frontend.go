// Package frontend embeds and serves the admin web interface.
package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

//go:generate npm --prefix ../../web install
//go:generate npm --prefix ../../web run build

//go:embed all:static
var assets embed.FS

// Handler returns an http.Handler that serves the embedded admin UI.
// Unknown paths (React client-side routes) fall back to index.html so
// that BrowserRouter navigation works when the user refreshes or bookmarks
// a sub-path like /agents or /history.
func Handler() http.Handler {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic("frontend: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// io/fs paths are slash-separated without a leading slash.
		fsPath := strings.TrimPrefix(r.URL.Path, "/")
		if fsPath == "" {
			fsPath = "."
		}
		f, openErr := sub.Open(fsPath)
		if openErr == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// Path not found — serve index.html and let the React router handle it.
		r2 := r.Clone(r.Context())
		r2.URL = &url.URL{Path: "/"}
		fileServer.ServeHTTP(w, r2)
	})
}
