package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"videoshare/middleware"
	"videoshare/model"
)

// AuthHandler handles password-based authentication for shared videos.
type AuthHandler struct {
	store *model.ResourceStore
	sm    *scs.SessionManager
}

// NewAuthHandler creates a new AuthHandler with injected dependencies.
func NewAuthHandler(store *model.ResourceStore, sm *scs.SessionManager) *AuthHandler {
	return &AuthHandler{store: store, sm: sm}
}

// AuthenticateAPI handles JSON password authentication for shared videos.
// POST /api/s/{id}/auth
func (h *AuthHandler) AuthenticateAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.store.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "Resource not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to load resource for auth", "id", id, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Global category videos are public — auto-auth and return redirect.
	if model.IsPublic(resource.CategoryID) {
		middleware.SetVideoAuth(r.Context(), h.sm)
		respondJSONOK(w, map[string]interface{}{
			"redirect": "/s/" + id + "/watch",
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(resource.PasswordHash), []byte(req.Password)); err != nil {
		respondJSONError(w, "Invalid password. Please try again.", http.StatusUnauthorized)
		return
	}

	// Mark the session as authenticated for video viewing.
	middleware.SetVideoAuth(r.Context(), h.sm)

	slog.Info("resource authenticated via API", "id", id)
	respondJSONOK(w, map[string]interface{}{
		"redirect": "/s/" + id + "/watch",
	})
}
