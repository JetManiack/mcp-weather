package restapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/auth"
	"github.com/JetManiack/mcp-weather/internal/storage"
)

type createActorRequest struct {
	Name string `json:"name"`
}

type issueCredentialResponse struct {
	Token string `json:"token"`
}

type actorResponse struct {
	storage.Actor
	HasActiveToken bool `json:"has_active_token"`
}

func listActorsHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actors, err := storage.ListAgents(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		active, err := storage.ActorsWithActiveToken(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		resp := make([]actorResponse, 0, len(actors))
		for _, a := range actors {
			resp = append(resp, actorResponse{Actor: a, HasActiveToken: active[a.ID]})
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func createActorHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createActorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("body must be a JSON object with name"))
			return
		}
		actor, err := storage.CreateAgent(db, req.Name)
		if errors.Is(err, storage.ErrEmptyName) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, actor)
	}
}

// deleteActorHandler revokes every credential instead of deleting the Actor
// row, so history stays attributable.
func deleteActorHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := storage.RevokeAllAgentCredentials(db, chi.URLParam(r, "id")); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func listCredentialsHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		creds, err := storage.ListAgentCredentials(db, chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, creds)
	}
}

func issueCredentialHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Issue(db, chi.URLParam(r, "id"), "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, issueCredentialResponse{Token: token})
	}
}

// revokeCredentialHandler revokes a credential by its own ID (DELETE /credentials/{id}).
func revokeCredentialHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		credID := chi.URLParam(r, "id")
		// Revoke by credential ID regardless of actor — look up actor via the credential.
		if err := storage.RevokeCredential(db, credID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
