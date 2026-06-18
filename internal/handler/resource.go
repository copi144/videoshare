package handler

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"videoshare/internal/model"
	"videoshare/internal/upload"
)

// ResourceHandler handles CRUD operations for video resources.
type ResourceHandler struct {
	store     *model.ResourceStore
	templates fs.FS
	dataDir   string
}

// NewResourceHandler creates a new ResourceHandler with injected dependencies.
func NewResourceHandler(store *model.ResourceStore, templates fs.FS, dataDir string) *ResourceHandler {
	return &ResourceHandler{store: store, templates: templates, dataDir: dataDir}
}

// List serves the admin page listing all uploaded videos.
// GET /admin
func (h *ResourceHandler) List(w http.ResponseWriter, r *http.Request) {
	resources, err := h.store.List()
	if err != nil {
		slog.Error("failed to list resources", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Failed to list resources")
		return
	}

	// Sanitize for display — clear sensitive data.
	for _, res := range resources {
		res.PasswordHash = ""
	}

	if err := parseAndRender(w, h.templates, "upload.html", &TemplateData{
		Title:     "Upload — VideoShare",
		Resources: resources,
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
		_ = parseAndRender(w, h.templates, "upload.html", &TemplateData{
			Title: "Upload — VideoShare",
			Error: err.Error(),
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

	resource := &model.Resource{
		ID:           id,
		Title:        title,
		Description:  description,
		PasswordHash: string(hash),
		Filename:     header.Filename,
		FileSize:     header.Size,
		ContentType:  contentType,
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

	// Path traversal prevention.
	videosDir := filepath.Clean(filepath.Join(h.dataDir, "videos"))
	filePath := filepath.Clean(filepath.Join(videosDir, id+".mp4"))
	if !strings.HasPrefix(filePath, videosDir) {
		respondError(w, r, http.StatusBadRequest, "Invalid path")
		return
	}

	err := h.store.DeleteWithFile(id, func() error {
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
