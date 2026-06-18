package handler

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"videoshare/internal/middleware"
	"videoshare/internal/model"
	"videoshare/internal/upload"
)

// ResourceHandler handles CRUD operations for video resources.
type ResourceHandler struct {
	store         *model.ResourceStore
	categoryStore *model.CategoryStore
	playlistStore *model.PlaylistStore
	templates     fs.FS
	dataDir       string
	sm            *scs.SessionManager
	userStore     *model.UserStore
}

// NewResourceHandler creates a new ResourceHandler with injected dependencies.
func NewResourceHandler(store *model.ResourceStore, categoryStore *model.CategoryStore,
	templates fs.FS, dataDir string,
	sm *scs.SessionManager, userStore *model.UserStore, playlistStore *model.PlaylistStore) *ResourceHandler {
	return &ResourceHandler{
		store:         store,
		categoryStore: categoryStore,
		playlistStore: playlistStore,
		templates:     templates,
		dataDir:       dataDir,
		sm:            sm,
		userStore:     userStore,
	}
}

// List serves the admin page listing uploaded videos.
// GET /admin
func (h *ResourceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)

	var resources []*model.Resource
	var err error
	if userRole == "admin" {
		resources, err = h.store.List()
	} else {
		resources, err = h.store.ListByUploader(userID)
	}
	if err != nil {
		slog.Error("failed to list resources", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Failed to list resources")
		return
	}

	// Sanitize for display — clear sensitive data.
	for _, res := range resources {
		res.PasswordHash = ""
	}

	// Load categories for the upload-form dropdown.
	var categories []*model.Category
	if userRole == "admin" {
		categories, err = h.categoryStore.List()
		if err != nil {
			slog.Error("failed to list categories", "error", err)
			categories = nil
		}
	} else {
		categories, err = h.categoryStore.ListByUploader(userID)
		if err != nil {
			slog.Error("failed to list categories by uploader", "error", err)
			categories = nil
		}
	}

	// Load all playlists for the add-to-playlist dropdown.
	var allPlaylists []*model.Playlist
	allPlaylists, err = h.playlistStore.ListAll()
	if err != nil {
		slog.Error("failed to list playlists", "error", err)
		allPlaylists = nil
	}

	// For each resource, load which playlists it belongs to.
	resourcePlaylists := make(map[string][]string)
	for _, res := range resources {
		playlistIDs, err := h.playlistStore.GetPlaylistsForResource(res.ID)
		if err != nil {
			slog.Error("failed to get playlists for resource", "resource_id", res.ID, "error", err)
			continue
		}
		resourcePlaylists[res.ID] = playlistIDs
	}

	// Build a lookup for playlist names.
	playlistNames := make(map[string]string)
	for _, pl := range allPlaylists {
		playlistNames[pl.ID] = pl.Name
	}

	// Determine which resources have no playlist membership.
	var unassigned []*model.Resource
	for _, res := range resources {
		if playlistIDs, ok := resourcePlaylists[res.ID]; !ok || len(playlistIDs) == 0 {
			unassigned = append(unassigned, res)
		}
	}

	username := middleware.GetUsername(r.Context(), h.sm)
	if err := parseAndRender(w, h.templates, "upload.html", &TemplateData{
		Title:      "Upload — VideoShare",
		Resources:  resources,
		IsLoggedIn: true,
		Username:   username,
		UserRole:   userRole,
		Data: map[string]interface{}{
			"Categories":        categories,
			"Playlists":         allPlaylists,
			"ResourcePlaylists": resourcePlaylists,
			"PlaylistNames":     playlistNames,
			"UnassignedVideos":  unassigned,
		},
	}); err != nil {
		slog.Error("failed to render upload template", "error", err)
	}
}

// Upload handles video file uploads.
// POST /api/upload
func (h *ResourceHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form: 500 MB max, 32 MB in-memory buffer.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		slog.Error("failed to parse multipart form", "error", err)
		respondError(w, r, http.StatusBadRequest, "Failed to parse form")
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	password := r.FormValue("password")
	categoryID := r.FormValue("category_id")

	file, header, err := r.FormFile("file")
	if err != nil {
		slog.Error("failed to get uploaded file", "error", err)
		respondError(w, r, http.StatusBadRequest, "Missing file")
		return
	}
	defer file.Close()

	// Validate the uploaded file at the boundary.
	userRole := middleware.GetUserRole(r.Context(), h.sm)
	username := middleware.GetUsername(r.Context(), h.sm)

	if err := upload.ValidateUpload(file, header); err != nil {
		slog.Error("upload validation failed", "error", err)
		// Load categories for the dropdown on re-render.
		var categories []*model.Category
		if userRole == "admin" {
			categories, _ = h.categoryStore.List()
		} else {
			categories, _ = h.categoryStore.ListByUploader(middleware.GetUserID(r.Context(), h.sm))
		}
		_ = parseAndRender(w, h.templates, "upload.html", &TemplateData{
			Title:      "Upload — VideoShare",
			Error:      err.Error(),
			IsLoggedIn: true,
			Username:   username,
			UserRole:   userRole,
			Data: map[string]interface{}{
				"Categories": categories,
			},
		})
		return
	}

	id := uuid.New().String()

	// Hash the share password — fail fast on error.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Failed to secure password")
		return
	}

	// Ensure the videos directory exists.
	videosDir := filepath.Join(h.dataDir, "videos")
	if err := os.MkdirAll(videosDir, 0o755); err != nil {
		slog.Error("failed to create videos directory", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}

	// Save the uploaded file to disk.
	dstPath := filepath.Join(videosDir, id+".mp4")
	dst, err := os.Create(dstPath)
	if err != nil {
		slog.Error("failed to create video file", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		slog.Error("failed to write video file", "error", err)
		os.Remove(dstPath)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}

	uploadedBy := middleware.GetUserID(r.Context(), h.sm)

	resource := &model.Resource{
		ID:           id,
		Title:        title,
		Description:  description,
		PasswordHash: string(hash),
		Filename:     header.Filename,
		FileSize:     header.Size,
		ContentType:  contentType,
		UploadedBy:   uploadedBy,
		CategoryID:   categoryID,
	}

	if err := h.store.Insert(resource); err != nil {
		slog.Error("failed to insert resource", "error", err)
		os.Remove(dstPath)
		respondError(w, r, http.StatusInternalServerError, "Failed to save record")
		return
	}

	slog.Info("resource uploaded",
		"id", id,
		"title", title,
		"size", header.Size,
	)

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// Delete removes a video resource and its file.
// DELETE /api/resource/{id}
func (h *ResourceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Look up the resource to check ownership before deleting.
	resource, err := h.store.GetByID(id)
	if err != nil {
		slog.Error("failed to load resource for deletion", "id", id, "error", err)
		respondError(w, r, http.StatusNotFound, "Resource not found")
		return
	}

	// Ownership check: admin can delete anything; uploader can delete own.
	userID := middleware.GetUserID(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)
	if userRole != "admin" && resource.UploadedBy != userID {
		respondError(w, r, http.StatusForbidden, "You can only delete your own videos")
		return
	}

	// Path traversal prevention.
	videosDir := filepath.Clean(filepath.Join(h.dataDir, "videos"))
	filePath := filepath.Clean(filepath.Join(videosDir, id+".mp4"))
	if !strings.HasPrefix(filePath, videosDir) {
		respondError(w, r, http.StatusBadRequest, "Invalid path")
		return
	}

	err = h.store.DeleteWithFile(id, func() error {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	})
	if err != nil {
		slog.Error("failed to delete resource", "id", id, "error", err)
		respondError(w, r, http.StatusInternalServerError, "Failed to delete resource")
		return
	}

	slog.Info("resource deleted", "id", id)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
