package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"videoshare/internal/middleware"
	"videoshare/internal/model"
)

// AuthHandler handles password-based authentication for shared videos.
type AuthHandler struct {
	store     *model.ResourceStore
	sm        *scs.SessionManager
	templates fs.FS
}

// NewAuthHandler creates a new AuthHandler with injected dependencies.
func NewAuthHandler(store *model.ResourceStore, sm *scs.SessionManager, templates fs.FS) *AuthHandler {
	return &AuthHandler{store: store, sm: sm, templates: templates}
}

// ServeSharePage renders the password entry form for a shared video.
// GET /s/{id}
func (h *AuthHandler) ServeSharePage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.store.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.Error("failed to load resource for share page", "id", id, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Global category videos are public — auto-auth and redirect to watch.
	if resource.CategoryID == model.GlobalCategoryID {
		middleware.SetVideoAuth(r.Context(), h.sm)
		http.Redirect(w, r, "/s/"+id+"/watch", http.StatusSeeOther)
		return
	}

	isLoggedIn := middleware.GetUserID(r.Context(), h.sm) != ""

	if err := parseAndRender(w, h.templates, "share.html", &TemplateData{
		Title:      "Enter Password — VideoShare",
		ResourceID: id,
		IsLoggedIn: isLoggedIn,
	}); err != nil {
		slog.Error("failed to render share template", "error", err)
	}
}

// Authenticate validates the provided password and grants session access.
// POST /s/{id}/auth
func (h *AuthHandler) Authenticate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.store.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.Error("failed to load resource for auth", "id", id, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Global category videos are public — auto-auth and redirect.
	if resource.CategoryID == model.GlobalCategoryID {
		middleware.SetVideoAuth(r.Context(), h.sm)
		http.Redirect(w, r, "/s/"+id+"/watch", http.StatusSeeOther)
		return
	}

	password := r.FormValue("password")

	if err := bcrypt.CompareHashAndPassword([]byte(resource.PasswordHash), []byte(password)); err != nil {
		// Password mismatch — re-render the share page with an error.
		isLoggedIn := middleware.GetUserID(r.Context(), h.sm) != ""
		if err := parseAndRender(w, h.templates, "share.html", &TemplateData{
			Title:      "Enter Password — VideoShare",
			ResourceID: id,
			Error:      "Invalid password. Please try again.",
			IsLoggedIn: isLoggedIn,
		}); err != nil {
			slog.Error("failed to render share template", "error", err)
		}
		return
	}

	// Mark the session as authenticated for video viewing.
	middleware.SetVideoAuth(r.Context(), h.sm)

	slog.Info("resource authenticated", "id", id)
	http.Redirect(w, r, "/s/"+id+"/watch", http.StatusSeeOther)
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
	if resource.CategoryID == model.GlobalCategoryID {
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

// ServeWatchPage displays the video player for an authenticated session.
// GET /s/{id}/watch
func (h *AuthHandler) ServeWatchPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.store.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.Error("failed to load resource for watch", "id", id, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Increment view count — best-effort, non-fatal.
	if err := h.store.IncrementViews(id); err != nil {
		slog.Error("failed to increment views", "id", id, "error", err)
	}

	username := middleware.GetUsername(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)
	isLoggedIn := middleware.GetUserID(r.Context(), h.sm) != ""

	if err := parseAndRender(w, h.templates, "watch.html", &TemplateData{
		Title:      resource.Title,
		IsLoggedIn: isLoggedIn,
		Username:   username,
		UserRole:   userRole,
		Data:       resource,
	}); err != nil {
		slog.Error("failed to render watch template", "error", err)
	}
}
