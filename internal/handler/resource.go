package handler

import (
	"io"
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
	dataDir       string
	sm            *scs.SessionManager
	userStore     *model.UserStore
}

// NewResourceHandler creates a new ResourceHandler with injected dependencies.
func NewResourceHandler(store *model.ResourceStore, categoryStore *model.CategoryStore,
	dataDir string,
	sm *scs.SessionManager, userStore *model.UserStore, playlistStore *model.PlaylistStore) *ResourceHandler {
	return &ResourceHandler{
		store:         store,
		categoryStore: categoryStore,
		playlistStore: playlistStore,
		dataDir:       dataDir,
		sm:            sm,
		userStore:     userStore,
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

	// Category is required.
	if categoryID == "" {
		respondError(w, r, http.StatusBadRequest, "Category is required")
		return
	}

	userID := middleware.GetUserID(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)

	// Non-admin users: verify authorization for non-global categories.
	if userRole != "admin" && categoryID != model.GlobalCategoryID {
		authorized, authErr := h.categoryStore.IsUploaderAuthorized(userID, categoryID)
		if authErr != nil {
			slog.Error("failed to check category authorization", "error", authErr)
			respondError(w, r, http.StatusInternalServerError, "Authorization error")
			return
		}
		if !authorized {
			respondError(w, r, http.StatusForbidden, "You are not authorized to upload to this category")
			return
		}
	}

	// Global category videos are public — no password needed.
	// All other categories require a password.
	var (
		hash []byte
		err  error
	)
	if categoryID == model.GlobalCategoryID {
		hash = nil // public video, no password hashing
	} else {
		if password == "" {
			respondError(w, r, http.StatusBadRequest, "Password is required for non-global categories")
			return
		}
		hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("failed to hash password", "error", err)
			respondError(w, r, http.StatusInternalServerError, "Failed to secure password")
			return
		}
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		slog.Error("failed to get uploaded file", "error", err)
		respondError(w, r, http.StatusBadRequest, "Missing file")
		return
	}
	defer file.Close()

	// Validate the uploaded file at the boundary.
	if err := upload.ValidateUpload(file, header); err != nil {
		slog.Error("upload validation failed", "error", err)
		respondJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := uuid.New().String()

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

	uploadedBy := userID

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

	respondJSONOK(w, map[string]interface{}{
		"redirect": "/admin",
	})
}

// ListResourcesAPI returns all resources as JSON (for populating dropdowns).
// GET /api/resources
func (h *ResourceHandler) ListResourcesAPI(w http.ResponseWriter, r *http.Request) {
	userRole := middleware.GetUserRole(r.Context(), h.sm)
	userID := middleware.GetUserID(r.Context(), h.sm)

	var resources []*model.Resource
	var err error
	if userRole == "admin" {
		resources, err = h.store.List()
	} else {
		resources, err = h.store.ListByUploader(userID)
	}
	if err != nil {
		slog.Error("failed to list resources", "error", err)
		respondJSONError(w, "Failed to list resources", http.StatusInternalServerError)
		return
	}

	// Sanitize for display.
	for _, res := range resources {
		res.PasswordHash = ""
	}

	respondJSONOK(w, map[string]interface{}{
		"resources": resources,
	})
}

// GetResourceAPI returns a single resource as JSON.
// GET /api/resources/{id}
func (h *ResourceHandler) GetResourceAPI(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, "Resource ID is required", http.StatusBadRequest)
		return
	}

	resource, err := h.store.GetByID(id)
	if err != nil {
		slog.Error("failed to get resource", "id", id, "error", err)
		respondJSONError(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Sanitize — never expose password hash.
	resource.PasswordHash = ""

	respondJSONOK(w, map[string]interface{}{
		"resource": resource,
	})
}

// DeleteResourceAPI removes a video resource and its file.
// DELETE /api/resource/{id}
func (h *ResourceHandler) DeleteResourceAPI(w http.ResponseWriter, r *http.Request) {
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

	slog.Info("resource deleted via API", "id", id)
	respondJSONOK(w, nil)
}
