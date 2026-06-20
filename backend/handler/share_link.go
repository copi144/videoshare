package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"videoshare/middleware"
	"videoshare/model"
)

// ShareLinkHandler handles CRUD for category/playlist share links.
type ShareLinkHandler struct {
	store         *model.ShareLinkStore
	categoryStore *model.CategoryStore
	playlistStore *model.PlaylistStore
}

// NewShareLinkHandler creates a new ShareLinkHandler.
func NewShareLinkHandler(store *model.ShareLinkStore, categoryStore *model.CategoryStore, playlistStore *model.PlaylistStore) *ShareLinkHandler {
	return &ShareLinkHandler{
		store:         store,
		categoryStore: categoryStore,
		playlistStore: playlistStore,
	}
}

// CreateAPI creates a new share link for a category or playlist.
// POST /api/share-links
func (h *ShareLinkHandler) CreateAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetType       string `json:"target_type"`
		TargetID         string `json:"target_id"`
		ExpiresInMinutes int    `json:"expires_in_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.TargetType != "category" && req.TargetType != "playlist" {
		respondJSONError(w, "target_type must be 'category' or 'playlist'", http.StatusBadRequest)
		return
	}
	if req.TargetID == "" {
		respondJSONError(w, "target_id is required", http.StatusBadRequest)
		return
	}
	if req.ExpiresInMinutes < 1 || req.ExpiresInMinutes > 525600 {
		respondJSONError(w, "Expiry must be between 1 minute and 365 days", http.StatusBadRequest)
		return
	}

	// Verify target exists
	if req.TargetType == "category" {
		if _, err := h.categoryStore.GetByName(req.TargetID); err != nil {
			respondJSONError(w, "Category not found", http.StatusNotFound)
			return
		}
	} else {
		if _, err := h.playlistStore.GetByID(req.TargetID); err != nil {
			respondJSONError(w, "Playlist not found", http.StatusNotFound)
			return
		}
	}

	id, err := model.GenerateShareLinkID()
	if err != nil {
		slog.Error("failed to generate share link ID", "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}
	password, err := model.GenerateShareLinkID()
	if err != nil {
		slog.Error("failed to generate share link password", "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	t := time.Now().UTC().Add(time.Duration(req.ExpiresInMinutes) * time.Minute)
	userID := middleware.GetUserIDFromContext(r.Context())
	now := time.Now().UTC()

	link := &model.ShareLink{
		ID:         id,
		Password:   password,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		ExpiresAt:  &t,
		CreatedBy:  userID,
		CreatedAt:  now,
	}

	if err := h.store.Create(link); err != nil {
		slog.Error("failed to create share link", "error", err)
		respondJSONError(w, "Failed to create share link", http.StatusInternalServerError)
		return
	}

	url := "/#/s/" + id + "/" + password

	slog.Info("share link created", "target_type", req.TargetType, "target_id", req.TargetID)
	respondJSONOK(w, map[string]interface{}{
		"ok":          true,
		"url":         url,
		"id":          id,
		"password":    password,
		"target_type": req.TargetType,
		"target_id":   req.TargetID,
		"expires_at":  t,
	})
}

// ListAPI lists share links for a target (category/playlist).
// GET /api/share-links?target_type=xxx&target_id=xxx
func (h *ShareLinkHandler) ListAPI(w http.ResponseWriter, r *http.Request) {
	targetType := r.URL.Query().Get("target_type")
	targetID := r.URL.Query().Get("target_id")
	if targetType == "" || targetID == "" {
		respondJSONError(w, "target_type and target_id are required", http.StatusBadRequest)
		return
	}

	links, err := h.store.ListByTarget(targetType, targetID)
	if err != nil {
		slog.Error("failed to list share links", "target_type", targetType, "target_id", targetID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	type linkInfo struct {
		ID         string     `json:"id"`
		TargetType string     `json:"target_type"`
		TargetID   string     `json:"target_id"`
		ExpiresAt  *time.Time `json:"expires_at"`
		CreatedBy  string     `json:"created_by"`
		CreatedAt  time.Time  `json:"created_at"`
	}
	var result []linkInfo
	for _, l := range links {
		result = append(result, linkInfo{
			ID:         l.ID,
			TargetType: l.TargetType,
			TargetID:   l.TargetID,
			ExpiresAt:  l.ExpiresAt,
			CreatedBy:  l.CreatedBy,
			CreatedAt:  l.CreatedAt,
		})
	}

	respondJSONOK(w, map[string]interface{}{"share_links": result})
}

// DeleteAPI deletes a share link by ID.
// DELETE /api/share-links/{id}
func (h *ShareLinkHandler) DeleteAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, "Share link ID is required", http.StatusBadRequest)
		return
	}
	if err := h.store.Delete(id); err != nil {
		slog.Error("failed to delete share link", "id", id, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}
	respondJSONOK(w, map[string]interface{}{"ok": true})
}

// AuthenticateAPI authenticates a share link via id + password (public endpoint).
// POST /api/share-links/{id}/auth
func (h *ShareLinkHandler) AuthenticateAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, "Share link ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		respondJSONError(w, "Password is required", http.StatusBadRequest)
		return
	}

	link, err := h.store.GetByID(id)
	if err != nil {
		respondJSONError(w, "Invalid or expired share link", http.StatusUnauthorized)
		return
	}

	if link.Password != req.Password {
		respondJSONError(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// Auth successful — determine redirect
	var redirect string
	if link.TargetType == "category" {
		redirect = "/#/admin/categories?category=" + link.TargetID
	} else {
		redirect = "/#/admin/playlists?playlist=" + link.TargetID
	}

	slog.Info("share link authenticated", "id", id, "target_type", link.TargetType, "target_id", link.TargetID)
	respondJSONOK(w, map[string]interface{}{
		"ok":          true,
		"redirect":    redirect,
		"target_type": link.TargetType,
		"target_id":   link.TargetID,
	})
}
