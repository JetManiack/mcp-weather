package restapi

import (
	"net/http"

	"github.com/JetManiack/mcp-weather/internal/humanauth"
)

func meHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, _ := humanauth.ActorFromContext(r.Context())
		role, _ := humanauth.RoleFromContext(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{
			"actor_id":     actor.ID,
			"display_name": actor.Name,
			"role":         role,
		})
	}
}
