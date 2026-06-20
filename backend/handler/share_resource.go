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

// ShareResourceHandler handles CRUD for single-resource share links.
type ShareResourceHandler struct {
	store         *model.ShareResourceStore
	resourceStore *model.ResourceStore
}

// NewShareResourceHandler creates a new ShareResourceHandler.
func NewShareResourceHandler(store *model.ShareResourceStore, resourceStore *model.ResourceStore) *ShareResourceHandler {
	return &ShareResourceHandler{store: store, resourceStore: resourceStore}
}

// CreateAPI creates a new share link for a resource.
// POST /api/share-resources
func (h *ShareResourceHandler) CreateAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceID       string `json:"resource_id"`
		ExpiresInMinutes int    `json:"expires_in_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.ResourceID == "" {
		respondJSONError(w, "Resource ID is required", http.StatusBadRequest)
		return
	}
	if req.ExpiresInMinutes < 1 || req.ExpiresInMinutes > 525600 {
		respondJSONError(w, "Expiry must be between 1 minute and 365 days", http.StatusBadRequest)
		return
	}

	// Verify resource exists
	if _, err := h.resourceStore.GetByID(req.ResourceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "Resource not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to lookup resource", "id", req.ResourceID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	password, err := model.GenerateSharePassword()
	if err != nil {
		slog.Error("failed to generate password", "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	t := time.Now().UTC().Add(time.Duration(req.ExpiresInMinutes) * time.Minute)
	userID := middleware.GetUserIDFromContext(r.Context())
	now := time.Now().UTC()

	link := &model.ShareResource{
		ResourceID: req.ResourceID,
		Password:   password,
		ExpiresAt:  &t,
		CreatedBy:  userID,
		CreatedAt:  now,
	}

	if err := h.store.Create(link); err != nil {
		slog.Error("failed to create share resource link", "error", err)
		respondJSONError(w, "Failed to create share link", http.StatusInternalServerError)
		return
	}

	url := "/#/v/" + req.ResourceID + "/" + password

	slog.Info("share resource link created", "resource_id", req.ResourceID, "expires_in_minutes", req.ExpiresInMinutes)
	respondJSONOK(w, map[string]interface{}{
		"ok":         true,
		"url":        url,
		"password":   password,
		"expires_at": t,
	})
}

// ListAPI lists share links for a resource.
// GET /api/share-resources?resource_id=xxx
func (h *ShareResourceHandler) ListAPI(w http.ResponseWriter, r *http.Request) {
	resourceID := r.URL.Query().Get("resource_id")
	if resourceID == "" {
		respondJSONError(w, "resource_id is required", http.StatusBadRequest)
		return
	}

	links, err := h.store.ListByResource(resourceID)
	if err != nil {
		slog.Error("failed to list share resource links", "resource_id", resourceID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	type linkInfo struct {
		ResourceID string     `json:"resource_id"`
		Password   string     `json:"password"`
		ExpiresAt  *time.Time `json:"expires_at"`
		CreatedBy  string     `json:"created_by"`
		CreatedAt  time.Time  `json:"created_at"`
	}
	var result []linkInfo
	for _, l := range links {
		result = append(result, linkInfo{
			ResourceID: l.ResourceID,
			Password:   l.Password,
			ExpiresAt:  l.ExpiresAt,
			CreatedBy:  l.CreatedBy,
			CreatedAt:  l.CreatedAt,
		})
	}

	respondJSONOK(w, map[string]interface{}{"share_links": result})
}

// DeleteAPI deletes a share resource link by resource ID and password.
// DELETE /api/share-resources/{resourceID}/{password}
func (h *ShareResourceHandler) DeleteAPI(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")
	password := chi.URLParam(r, "password")
	if resourceID == "" || password == "" {
		respondJSONError(w, "Resource ID and password are required", http.StatusBadRequest)
		return
	}

	if err := h.store.Delete(resourceID, password); err != nil {
		slog.Error("failed to delete share resource link", "resource_id", resourceID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}
	respondJSONOK(w, map[string]interface{}{"ok": true})
}
