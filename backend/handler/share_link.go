package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"videoshare/middleware"
	"videoshare/model"
)

// ShareLinkHandler handles CRUD for ephemeral share links.
type ShareLinkHandler struct {
	shareLinkStore *model.ShareLinkStore
	resourceStore  *model.ResourceStore
}

// NewShareLinkHandler creates a new ShareLinkHandler.
func NewShareLinkHandler(shareLinkStore *model.ShareLinkStore, resourceStore *model.ResourceStore) *ShareLinkHandler {
	return &ShareLinkHandler{
		shareLinkStore: shareLinkStore,
		resourceStore:  resourceStore,
	}
}

// CreateShareLinkAPI creates a new share link for a resource.
// POST /api/share-links
func (h *ShareLinkHandler) CreateShareLinkAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceID    string `json:"resource_id"`
		ExpiresInDays int    `json:"expires_in_days"` // 0 means no expiry
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ResourceID == "" {
		respondJSONError(w, "Resource ID is required", http.StatusBadRequest)
		return
	}

	// Verify resource exists
	resource, err := h.resourceStore.GetByID(req.ResourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "Resource not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to lookup resource", "id", req.ResourceID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}
	_ = resource // resource exists, proceed

	// Generate ID and password
	id, err := model.GenerateShareLinkID()
	if err != nil {
		slog.Error("failed to generate share link ID", "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}
	password, err := model.GenerateShareLinkPassword()
	if err != nil {
		slog.Error("failed to generate password", "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Compute expiry
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		t := time.Now().UTC().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}

	userID := middleware.GetUserIDFromContext(r.Context())
	now := time.Now().UTC()

	link := &model.ShareLink{
		ID:         id,
		ResourceID: req.ResourceID,
		Password:   password,
		ExpiresAt:  expiresAt,
		CreatedBy:  userID,
		CreatedAt:  now,
	}

	if err := h.shareLinkStore.Create(link); err != nil {
		slog.Error("failed to create share link", "error", err)
		respondJSONError(w, "Failed to create share link", http.StatusInternalServerError)
		return
	}

	// Build the share URL
	url := "/#/v/" + req.ResourceID + "/" + password

	slog.Info("share link created", "resource_id", req.ResourceID, "expires_in_days", req.ExpiresInDays)
	respondJSONOK(w, map[string]interface{}{
		"ok":         true,
		"url":        url,
		"id":         id,
		"password":   password,
		"expires_at": expiresAt,
	})
}

// ListShareLinksAPI lists share links for a resource.
// GET /api/share-links?resource_id=xxx
func (h *ShareLinkHandler) ListShareLinksAPI(w http.ResponseWriter, r *http.Request) {
	resourceID := r.URL.Query().Get("resource_id")
	if resourceID == "" {
		respondJSONError(w, "resource_id is required", http.StatusBadRequest)
		return
	}

	links, err := h.shareLinkStore.ListByResource(resourceID)
	if err != nil {
		slog.Error("failed to list share links", "resource_id", resourceID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Don't expose passwords in listing
	type linkInfo struct {
		ID         string     `json:"id"`
		ResourceID string     `json:"resource_id"`
		ExpiresAt  *time.Time `json:"expires_at"`
		CreatedBy  string     `json:"created_by"`
		CreatedAt  time.Time  `json:"created_at"`
	}
	var result []linkInfo
	for _, l := range links {
		result = append(result, linkInfo{
			ID:         l.ID,
			ResourceID: l.ResourceID,
			ExpiresAt:  l.ExpiresAt,
			CreatedBy:  l.CreatedBy,
			CreatedAt:  l.CreatedAt,
		})
	}

	respondJSONOK(w, map[string]interface{}{
		"share_links": result,
	})
}

// DeleteShareLinkAPI deletes a share link.
// DELETE /api/share-links/{id}
func (h *ShareLinkHandler) DeleteShareLinkAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, "Share link ID is required", http.StatusBadRequest)
		return
	}

	if err := h.shareLinkStore.Delete(id); err != nil {
		slog.Error("failed to delete share link", "id", id, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	respondJSONOK(w, map[string]interface{}{"ok": true})
}
