package restapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/storage"
)

type historyListResponse struct {
	Calls      []storage.ToolCall `json:"calls"`
	NextCursor string             `json:"next_cursor,omitempty"`
	// Actors maps actor IDs → display names so the UI can label rows without
	// a separate lookup per agent.
	Actors map[string]string `json:"actors"`
}

func listHistoryHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit := 0
		if raw := q.Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, errors.New("limit must be an integer"))
				return
			}
			limit = parsed
		}
		if status := q.Get("status"); status != "" &&
			status != string(storage.ToolCallStatusOK) && status != string(storage.ToolCallStatusError) {
			writeError(w, http.StatusBadRequest, errors.New(`status must be "ok" or "error"`))
			return
		}

		calls, next, err := storage.ListToolCalls(db, storage.HistoryFilter{
			ActorID: q.Get("actor"),
			Tool:    q.Get("tool"),
			Status:  q.Get("status"),
			Limit:   limit,
			Cursor:  q.Get("cursor"),
		})
		if errors.Is(err, storage.ErrUnknownCursor) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		actorIDs := make([]string, 0, len(calls))
		for _, c := range calls {
			actorIDs = append(actorIDs, c.ActorID)
		}
		names, err := storage.ActorNamesByID(db, actorIDs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, historyListResponse{Calls: calls, NextCursor: next, Actors: names})
	}
}

func getHistoryEntryHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		call, err := storage.GetToolCall(db, chi.URLParam(r, "id"))
		if errors.Is(err, storage.ErrToolCallNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		names, err := storage.ActorNamesByID(db, []string{call.ActorID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"call": call, "actor": names[call.ActorID]})
	}
}

func listHistoryToolsHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tools, err := storage.DistinctTools(db)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, tools)
	}
}
