package handler

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"videoshare/internal/middleware"
	"videoshare/internal/model"
)

// PlaylistHandler handles playlist management (admin only).
type PlaylistHandler struct {
	playlistStore  *model.PlaylistStore
	resourceStore  *model.ResourceStore
	categoryStore  *model.CategoryStore
	sm             *scs.SessionManager
	templates      fs.FS
}

// NewPlaylistHandler creates a new PlaylistHandler with injected dependencies.
func NewPlaylistHandler(
	playlistStore *model.PlaylistStore,
	resourceStore *model.ResourceStore,
	categoryStore *model.CategoryStore,
	sm *scs.SessionManager,
	templates fs.FS,
) *PlaylistHandler {
	return &PlaylistHandler{
		playlistStore: playlistStore,
		resourceStore: resourceStore,
		categoryStore: categoryStore,
		sm:            sm,
		templates:     templates,
	}
}

// ServePlaylistsPage lists all playlists grouped by category.
// GET /admin/playlists
func (h *PlaylistHandler) ServePlaylistsPage(w http.ResponseWriter, r *http.Request) {
	username := middleware.GetUsername(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)

	categories, err := h.categoryStore.List()
	if err != nil {
		slog.Error("failed to list categories", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Build map: categoryID -> []*Playlist
	playlistMap := make(map[string][]*model.Playlist)
	// Build map: playlistID -> []*Resource
	videoMap := make(map[string][]*model.Resource)

	for _, cat := range categories {
		playlists, err := h.playlistStore.ListByCategory(cat.ID)
		if err != nil {
			slog.Error("failed to list playlists for category", "category_id", cat.ID, "error", err)
			continue
		}
		playlistMap[cat.ID] = playlists

		for _, pl := range playlists {
			videoIDs, err := h.playlistStore.ListVideos(pl.ID)
			if err != nil {
				slog.Error("failed to list videos for playlist", "playlist_id", pl.ID, "error", err)
				continue
			}
			var resources []*model.Resource
			for _, vidID := range videoIDs {
				res, err := h.resourceStore.GetByID(vidID)
				if err != nil {
					slog.Error("failed to get resource", "resource_id", vidID, "error", err)
					continue
				}
				res.PasswordHash = ""
				resources = append(resources, res)
			}
			videoMap[pl.ID] = resources
		}
	}

	// Load all resources for the add-video dropdown.
	allVideos, err := h.resourceStore.List()
	if err != nil {
		slog.Error("failed to list resources", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	for _, res := range allVideos {
		res.PasswordHash = ""
	}

	errorMsg := r.URL.Query().Get("error")

	if err := parseAndRender(w, h.templates, "playlists.html", &TemplateData{
		Title:      "Playlists — VideoShare",
		IsLoggedIn: true,
		Username:   username,
		UserRole:   userRole,
		Error:      errorMsg,
		Data: map[string]interface{}{
			"Categories": categories,
			"Playlists":  playlistMap,
			"Videos":     videoMap,
			"AllVideos":  allVideos,
		},
	}); err != nil {
		slog.Error("failed to render playlists template", "error", err)
	}
}

// CreatePlaylist creates a new playlist.
// POST /admin/playlists
func (h *PlaylistHandler) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	description := r.FormValue("description")
	categoryID := r.FormValue("category_id")

	if name == "" {
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Playlist name is required"), http.StatusSeeOther)
		return
	}
	if categoryID == "" {
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Category is required"), http.StatusSeeOther)
		return
	}

	sortOrderStr := r.FormValue("sort_order")
	sortOrder := 0
	if sortOrderStr != "" {
		var err error
		sortOrder, err = strconv.Atoi(sortOrderStr)
		if err != nil {
			sortOrder = 0
		}
	}

	userID := middleware.GetUserID(r.Context(), h.sm)

	pl := &model.Playlist{
		ID:          uuid.New().String(),
		CategoryID:  categoryID,
		Name:        name,
		Description: description,
		CreatedBy:   userID,
		SortOrder:   sortOrder,
	}

	if err := h.playlistStore.Insert(pl); err != nil {
		slog.Error("failed to create playlist", "error", err)
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Failed to create playlist"), http.StatusSeeOther)
		return
	}

	slog.Info("playlist created", "id", pl.ID, "name", name, "category_id", categoryID)
	http.Redirect(w, r, "/admin/playlists", http.StatusSeeOther)
}

// DeletePlaylist deletes a playlist.
// POST /admin/playlists/{id}/delete
func (h *PlaylistHandler) DeletePlaylist(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Missing playlist ID"), http.StatusSeeOther)
		return
	}

	if err := h.playlistStore.Delete(id); err != nil {
		slog.Error("failed to delete playlist", "id", id, "error", err)
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Failed to delete playlist"), http.StatusSeeOther)
		return
	}

	slog.Info("playlist deleted", "id", id)
	http.Redirect(w, r, "/admin/playlists", http.StatusSeeOther)
}

// AddVideoToPlaylist adds a video to a playlist.
// POST /admin/playlists/{id}/videos
func (h *PlaylistHandler) AddVideoToPlaylist(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Missing playlist ID"), http.StatusSeeOther)
		return
	}

	resourceID := r.FormValue("resource_id")
	if resourceID == "" {
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Missing resource ID"), http.StatusSeeOther)
		return
	}

	if err := h.playlistStore.AddVideo(playlistID, resourceID, 0); err != nil {
		slog.Error("failed to add video to playlist", "playlist_id", playlistID, "resource_id", resourceID, "error", err)
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Failed to add video"), http.StatusSeeOther)
		return
	}

	slog.Info("video added to playlist", "playlist_id", playlistID, "resource_id", resourceID)
	http.Redirect(w, r, "/admin/playlists", http.StatusSeeOther)
}

// RemoveVideoFromPlaylist removes a video from a playlist.
// POST /admin/playlists/{id}/videos/remove
func (h *PlaylistHandler) RemoveVideoFromPlaylist(w http.ResponseWriter, r *http.Request) {
	playlistID := chi.URLParam(r, "id")
	if playlistID == "" {
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Missing playlist ID"), http.StatusSeeOther)
		return
	}

	resourceID := r.FormValue("resource_id")
	if resourceID == "" {
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Missing resource ID"), http.StatusSeeOther)
		return
	}

	if err := h.playlistStore.RemoveVideo(playlistID, resourceID); err != nil {
		slog.Error("failed to remove video from playlist", "playlist_id", playlistID, "resource_id", resourceID, "error", err)
		http.Redirect(w, r, "/admin/playlists?error="+url.QueryEscape("Failed to remove video"), http.StatusSeeOther)
		return
	}

	slog.Info("video removed from playlist", "playlist_id", playlistID, "resource_id", resourceID)
	http.Redirect(w, r, "/admin/playlists", http.StatusSeeOther)
}

// CreatePlaylistAPI handles JSON playlist creation.
// POST /api/playlists
func (h *PlaylistHandler) CreatePlaylistAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CategoryID  string `json:"category_id"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondJSONError(w, "Playlist name is required", http.StatusBadRequest)
		return
	}
	if req.CategoryID == "" {
		respondJSONError(w, "Category is required", http.StatusBadRequest)
		return
	}

	userID := middleware.GetUserID(r.Context(), h.sm)

	pl := &model.Playlist{
		ID:          uuid.New().String(),
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
		SortOrder:   req.SortOrder,
	}

	if err := h.playlistStore.Insert(pl); err != nil {
		slog.Error("failed to create playlist", "error", err)
		respondJSONError(w, "Failed to create playlist", http.StatusInternalServerError)
		return
	}

	slog.Info("playlist created via API", "id", pl.ID, "name", req.Name, "category_id", req.CategoryID)
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
	playlists, err := h.playlistStore.ListAll()
	if err != nil {
		slog.Error("failed to list playlists", "error", err)
		respondJSONError(w, "Failed to list playlists", http.StatusInternalServerError)
		return
	}

	respondJSONOK(w, map[string]interface{}{
		"playlists": playlists,
	})
}
