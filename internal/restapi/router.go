// Package restapi is the human-facing REST API for managing agent tokens
// and browsing the audit log.
package restapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// Handler builds the REST API router. If adminToken is non-empty every
// route requires Authorization: Bearer <adminToken> (constant-time check).
// Optional domain functions may mount additional routes on the router.
func Handler(db *gorm.DB, adminToken string, domain ...func(chi.Router)) http.Handler {
	r := chi.NewRouter()
	if adminToken != "" {
		r.Use(requireAdminToken(adminToken))
	}

	r.Route("/actors", func(r chi.Router) {
		r.Get("/", listActorsHandler(db))
		r.Post("/", createActorHandler(db))
		r.Delete("/{id}", deleteActorHandler(db))
		r.Get("/{id}/credentials", listCredentialsHandler(db))
		r.Post("/{id}/credentials", issueCredentialHandler(db))
	})

	r.Delete("/credentials/{id}", revokeCredentialHandler(db))

	r.Route("/tool-calls", func(r chi.Router) {
		r.Get("/", listToolCallsHandler(db))
		r.Get("/tools", listToolCallToolsHandler(db))
		r.Get("/{id}", getToolCallHandler(db))
	})

	for _, mount := range domain {
		mount(r)
	}

	return r
}

func requireAdminToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="weather-admin"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
