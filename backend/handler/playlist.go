package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"videoshare/middleware"
	"videoshare/model"
)

// PlaylistHandler handles playlist management (admin only).
type PlaylistHandler struct {
	playlistStore  *model.PlaylistStore
	resourceStore  *model.ResourceStore
	categoryStore  *model.CategoryStore
	sm             *scs.SessionManager
}

// NewPlaylistHandler creates a new PlaylistHandler with injected dependencies.
func NewPlaylistHandler(
	playlistStore *model.PlaylistStore,
	resourceStore *model.ResourceStore,
	categoryStore *model.CategoryStore,
	sm *scs.SessionManager,
) *PlaylistHandler {
	return &PlaylistHandler{
		playlistStore: playlistStore,
		resourceStore: resourceStore,
		categoryStore: categoryStore,
		sm:            sm,
	}
}

// CreatePlaylistAPI handles JSON playlist creation.
// POST /api/playlists
func (h *PlaylistHandler) CreatePlaylistAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		CategoryID   string `json:"category_id"`
		SortOrder    int    `json:"sort_order"`
		PlaylistType string `json:"playlist_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondJSONError(w, "Playlist name is required", http.StatusBadRequest)
		return
	}
	if !model.IsValidName(req.Name) {
		respondJSONError(w, "Playlist name must only contain letters, numbers, and hyphens", http.StatusBadRequest)
		return
	}
	if req.CategoryID == "" {
		respondJSONError(w, "Category is required", http.StatusBadRequest)
		return
	}

	// Default to "video" if type is empty, then validate.
	if req.PlaylistType == "" {
		req.PlaylistType = "video"
	}
	if req.PlaylistType != "video" && req.PlaylistType != "audio" && req.PlaylistType != "image" {
		respondJSONError(w, "Invalid playlist type", http.StatusBadRequest)
		return
	}

	userID := middleware.GetUserID(r.Context(), h.sm)

	pl := &model.Playlist{
		ID:           uuid.New().String(),
		CategoryName: req.CategoryID,
		PlaylistType: req.PlaylistType,
		Name:         req.Name,
		Description:  req.Description,
		CreatedBy:    userID,
		SortOrder:    req.SortOrder,
	}

	if err := h.playlistStore.Insert(pl); err != nil {
		slog.Error("failed to create playlist", "error", err)
		respondJSONError(w, "Failed to create playlist", http.StatusInternalServerError)
		return
	}

	slog.Info("playlist created via API", "id", pl.ID, "name", req.Name, "category_name", req.CategoryID, "playlist_type", req.PlaylistType)
	respondJSONOK(w, map[string]interface{}{
		"redirect": "/admin/playlists",
	})
}

// DeletePlaylistAPI handles JSON playlist deletion.
// DELETE /api/playlists/{id}
func (h *PlaylistHandler) DeletePlaylistAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, "Missing playlist ID", http.StatusBadRequest)
		return
	}

	if err := h.playlistStore.Delete(id); err != nil {
		slog.Error("failed to delete playlist", "id", id, "error", err)
		respondJSONError(w, "Failed to delete playlist", http.StatusInternalServerError)
		return
	}

	slog.Info("playlist deleted via API", "id", id)
	respondJSONOK(w, nil)
}

// AddVideoAPI handles JSON add-video-to-playlist.
// POST /api/playlists/{id}/videos
func (h *PlaylistHandler) AddVideoAPI(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		respondJSONError(w, "Missing playlist ID", http.StatusBadRequest)
		return
	}

	var req struct {
		ResourceID string `json:"resource_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResourceID == "" {
		respondJSONError(w, "Missing resource ID", http.StatusBadRequest)
		return
	}

	if err := h.playlistStore.AddVideo(playlistID, req.ResourceID, 0); err != nil {
		slog.Error("failed to add video to playlist", "playlist_id", playlistID, "resource_id", req.ResourceID, "error", err)
		respondJSONError(w, "Failed to add video", http.StatusInternalServerError)
		return
	}

	slog.Info("video added to playlist via API", "playlist_id", playlistID, "resource_id", req.ResourceID)
	respondJSONOK(w, nil)
}

// RemoveVideoAPI handles JSON remove-video-from-playlist.
// DELETE /api/playlists/{id}/videos/{resourceId}
func (h *PlaylistHandler) RemoveVideoAPI(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		respondJSONError(w, "Missing playlist ID", http.StatusBadRequest)
		return
	}

	resourceID := chi.URLParam(r, "resourceId")
	if resourceID == "" {
		respondJSONError(w, "Missing resource ID", http.StatusBadRequest)
		return
	}

	if err := h.playlistStore.RemoveVideo(playlistID, resourceID); err != nil {
		slog.Error("failed to remove video from playlist", "playlist_id", playlistID, "resource_id", resourceID, "error", err)
		respondJSONError(w, "Failed to remove video", http.StatusInternalServerError)
		return
	}

	slog.Info("video removed from playlist via API", "playlist_id", playlistID, "resource_id", resourceID)
	respondJSONOK(w, nil)
}

// ListPlaylistsAPI returns all playlists as JSON.
// GET /api/playlists
func (h *PlaylistHandler) ListPlaylistsAPI(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters at the boundary.
	const defaultLimit = 50
	const maxLimit = 100

	limit := defaultLimit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l <= 0 {
				limit = defaultLimit
			} else if l > maxLimit {
				limit = maxLimit
			} else {
				limit = l
			}
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			if o < 0 {
				offset = 0
			} else {
				offset = o
			}
		}
	}

	var playlists []*model.Playlist
	var total int
	var err error
	playlistType := r.URL.Query().Get("playlist_type")
	if playlistType != "" {
		playlists, err = h.playlistStore.ListByTypePaginated(playlistType, limit, offset)
		if err == nil {
			total, err = h.playlistStore.CountByType(playlistType)
		}
	} else {
		playlists, err = h.playlistStore.ListPaginated(limit, offset)
		if err == nil {
			total, err = h.playlistStore.Count()
		}
	}
	if err != nil {
		slog.Error("failed to list playlists", "error", err)
		respondJSONError(w, "Failed to list playlists", http.StatusInternalServerError)
		return
	}

	respondJSONOK(w, map[string]interface{}{
		"playlists": playlists,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}
