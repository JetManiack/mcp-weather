// Package restapi is the human-facing REST API for managing agent tokens
// and browsing the audit log.
package restapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// NewHandler builds the REST API router. If adminToken is non-empty every
// route requires Authorization: Bearer <adminToken>.
func NewHandler(db *gorm.DB, adminToken string) http.Handler {
	r := chi.NewRouter()
	if adminToken != "" {
		r.Use(requireAdminToken(adminToken))
	}

	r.Route("/agents", func(r chi.Router) {
		r.Get("/", listAgentsHandler(db))
		r.Post("/", createAgentHandler(db))
		r.Delete("/{id}", deleteAgentHandler(db))
		r.Get("/{id}/tokens", listAgentTokensHandler(db))
		r.Post("/{id}/tokens", issueTokenHandler(db))
		r.Delete("/{id}/tokens/{tokenID}", revokeTokenHandler(db))
	})

	r.Route("/history", func(r chi.Router) {
		r.Get("/", listHistoryHandler(db))
		r.Get("/tools", listHistoryToolsHandler(db))
		r.Get("/{id}", getHistoryEntryHandler(db))
	})

	return r
}

func requireAdminToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || got != token {
				w.Header().Set("WWW-Authenticate", `Bearer realm="weather-admin"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
