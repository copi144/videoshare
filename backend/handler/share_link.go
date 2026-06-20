package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"

	"videoshare/middleware"
	"videoshare/model"
)

// ShareLinkHandler handles CRUD for category/playlist share links.
type ShareLinkHandler struct {
	store         *model.ShareLinkStore
	categoryStore *model.CategoryStore
	playlistStore *model.PlaylistStore
	resourceStore *model.ResourceStore
	sm            *scs.SessionManager
}

// NewShareLinkHandler creates a new ShareLinkHandler.
func NewShareLinkHandler(store *model.ShareLinkStore, categoryStore *model.CategoryStore, playlistStore *model.PlaylistStore, resourceStore *model.ResourceStore, sm *scs.SessionManager) *ShareLinkHandler {
	return &ShareLinkHandler{
		store:         store,
		categoryStore: categoryStore,
		playlistStore: playlistStore,
		resourceStore: resourceStore,
		sm:            sm,
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
		if _, err := h.playlistStore.GetByNameOnly(req.TargetID); err != nil {
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
		ID:        id,
		Password:  password,
		TargetType: req.TargetType,
		ExpiresAt: &t,
		CreatedBy: userID,
		CreatedAt: now,
	}
	link.TargetID = req.TargetID

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

	// Auth successful — set session auth and store scope
	middleware.SetVideoAuth(r.Context(), h.sm)
	h.sm.Put(r.Context(), "share_target_type", link.TargetType)
	h.sm.Put(r.Context(), "share_target_id", link.TargetID)
	var redirect string
	if link.TargetType == "category" {
		redirect = "/#/c/" + link.TargetID
	} else {
		redirect = "/#/l/" + link.TargetID
	}

	slog.Info("share link authenticated", "id", id, "target_type", link.TargetType, "target_id", link.TargetID)
	respondJSONOK(w, map[string]interface{}{
		"ok":          true,
		"redirect":    redirect,
		"target_type": link.TargetType,
		"target_id":   link.TargetID,
		"target_name": link.TargetID,
	})
}

// GetSharedResourcesAPI returns resources for a share link (public, password-protected).
// GET /api/share-links/{id}/resources?password=xxx[&playlist_name=xxx]
func (h *ShareLinkHandler) GetSharedResourcesAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	password := r.URL.Query().Get("password")
	if id == "" || password == "" {
		respondJSONError(w, "Missing share link ID or password", http.StatusBadRequest)
		return
	}

	link, err := h.store.GetByID(id)
	if err != nil {
		respondJSONError(w, "Invalid share link", http.StatusUnauthorized)
		return
	}
	if link.Password != password {
		respondJSONError(w, "Invalid password", http.StatusUnauthorized)
		return
	}
	if link.ExpiresAt != nil && time.Now().UTC().After(*link.ExpiresAt) {
		respondJSONError(w, "Share link has expired", http.StatusUnauthorized)
		return
	}

	type resourceInfo struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Filename     string `json:"filename"`
		FileSize     int64  `json:"file_size"`
		ContentType  string `json:"content_type"`
		ResourceType string `json:"resource_type"`
		Views        int    `json:"views"`
		CreatedAt    string `json:"created_at"`
	}
	type playlistInfo struct {
		Name         string `json:"name"`
		DisplayName  string `json:"display_name"`
		PlaylistType string `json:"playlist_type"`
	}

	const limit = 100
	const offset = 0

	if link.TargetType == "category" {
		category, err := h.categoryStore.GetByName(link.TargetID)
		if err != nil {
			slog.Error("category not found for share link", "category", link.TargetID, "error", err)
			respondJSONError(w, "Target category not found", http.StatusNotFound)
			return
		}

		// Load playlists in this category
		playlists, err := h.playlistStore.ListByCategory(link.TargetID)
		if err != nil {
			slog.Error("failed to list playlists for share link", "error", err)
			// Non-fatal — proceed without playlists
			playlists = nil
		}
		plInfo := make([]playlistInfo, 0, len(playlists))
		for _, pl := range playlists {
			plInfo = append(plInfo, playlistInfo{
				Name:         pl.Name,
				DisplayName:  pl.DisplayName,
				PlaylistType: pl.PlaylistType,
			})
		}

		// Check if a specific playlist is requested
		playlistName := r.URL.Query().Get("playlist_name")
		if playlistName != "" {
			// Only return videos in that playlist
			playlist, err := h.playlistStore.GetByNameOnly(playlistName)
			if err != nil {
				respondJSONError(w, "Playlist not found", http.StatusNotFound)
				return
			}
			videoIDs, err := h.playlistStore.ListVideos(playlist.CategoryName, playlist.Name)
			if err != nil {
				slog.Error("failed to list playlist videos", "error", err)
				respondJSONError(w, "Failed to list videos", http.StatusInternalServerError)
				return
			}
			items := make([]resourceInfo, 0, len(videoIDs))
			for _, vid := range videoIDs {
				res, err := h.resourceStore.GetByID(vid)
				if err != nil || res.Banned {
					continue
				}
				items = append(items, resourceInfo{
					ID:           res.ID,
					Title:        res.Title,
					Filename:     res.Filename,
					FileSize:     res.FileSize,
					ContentType:  res.ContentType,
					ResourceType: res.ResourceType,
					Views:        res.Views,
					CreatedAt:    res.CreatedAt.Format(time.RFC3339),
				})
			}
			respondJSONOK(w, map[string]interface{}{
				"ok":          true,
				"target_type": "category",
				"target_name": category.DisplayName,
				"resources":   items,
				"playlists":   plInfo,
			})
			return
		}

		// No playlist filter — return all resources in the category
		resources, err := h.resourceStore.ListByCategoryPaginated(link.TargetID, limit, offset)
		if err != nil {
			slog.Error("failed to list resources for share link", "error", err)
			respondJSONError(w, "Failed to list resources", http.StatusInternalServerError)
			return
		}
		items := make([]resourceInfo, 0, len(resources))
		for _, res := range resources {
			if res.Banned {
				continue
			}
			items = append(items, resourceInfo{
				ID:           res.ID,
				Title:        res.Title,
				Filename:     res.Filename,
				FileSize:     res.FileSize,
				ContentType:  res.ContentType,
				ResourceType: res.ResourceType,
				Views:        res.Views,
				CreatedAt:    res.CreatedAt.Format(time.RFC3339),
			})
		}
		respondJSONOK(w, map[string]interface{}{
			"ok":          true,
			"target_type": "category",
			"target_name": category.DisplayName,
			"resources":   items,
			"playlists":   plInfo,
		})
		return
	}

	if link.TargetType == "playlist" {
		playlist, err := h.playlistStore.GetByNameOnly(link.TargetID)
		if err != nil {
			slog.Error("playlist not found for share link", "playlist", link.TargetID, "error", err)
			respondJSONError(w, "Target playlist not found", http.StatusNotFound)
			return
		}
		videoIDs, err := h.playlistStore.ListVideos(playlist.CategoryName, playlist.Name)
		if err != nil {
			slog.Error("failed to list playlist videos", "error", err)
			respondJSONError(w, "Failed to list videos", http.StatusInternalServerError)
			return
		}
		items := make([]resourceInfo, 0, len(videoIDs))
		for _, vid := range videoIDs {
			res, err := h.resourceStore.GetByID(vid)
			if err != nil {
				continue // skip missing resources
			}
			if res.Banned {
				continue
			}
			items = append(items, resourceInfo{
				ID:           res.ID,
				Title:        res.Title,
				Filename:     res.Filename,
				FileSize:     res.FileSize,
				ContentType:  res.ContentType,
				ResourceType: res.ResourceType,
				Views:        res.Views,
				CreatedAt:    res.CreatedAt.Format(time.RFC3339),
			})
		}
		respondJSONOK(w, map[string]interface{}{
			"ok":          true,
			"target_type": "playlist",
			"target_name": playlist.DisplayName,
			"resources":   items,
		})
		return
	}

	respondJSONError(w, "Invalid target type", http.StatusBadRequest)
}
