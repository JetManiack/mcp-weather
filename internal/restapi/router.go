// Package restapi is the human-facing REST API for managing agent tokens
// and browsing the audit log.
package restapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/humanauth"
)

// NewHandler builds the REST API router, requiring human authentication on
// every route. Admin-only routes additionally check for role == "admin".
func NewHandler(db *gorm.DB, provider humanauth.Provider) http.Handler {
	r := chi.NewRouter()
	r.Use(humanauth.RequireHumanAuth(db, provider))

	// Any authenticated human can view tool-call history and their own profile.
	r.Get("/me", meHandler())

	r.Route("/tool-calls", func(r chi.Router) {
		r.Get("/", listToolCallsHandler(db))
		r.Get("/tools", listToolCallToolsHandler(db))
		r.Get("/{id}", getToolCallHandler(db))
	})

	// Admin-only: actor and credential management.
	r.Group(func(r chi.Router) {
		r.Use(humanauth.RequireAdmin)
		r.Route("/actors", func(r chi.Router) {
			r.Get("/", listActorsHandler(db))
			r.Post("/", createActorHandler(db))
			r.Delete("/{id}", deleteActorHandler(db))
			r.Get("/{id}/credentials", listCredentialsHandler(db))
			r.Post("/{id}/credentials", issueCredentialHandler(db))
		})
		r.Delete("/credentials/{id}", revokeCredentialHandler(db))
	})

	return r
}
