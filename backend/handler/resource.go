package handler

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"lukechampine.com/blake3"

	"videoshare/middleware"
	"videoshare/model"
	"videoshare/storage"
	"videoshare/transcode"
	"videoshare/upload"
)

// ResourceHandler handles CRUD operations for video resources.
type ResourceHandler struct {
	store           *model.ResourceStore
	categoryStore   *model.CategoryStore
	playlistStore   *model.PlaylistStore
	transcodeQueue  *transcode.Queue
	dataDir         string
	sm              *scs.SessionManager
	userStore       *model.UserStore
}

// NewResourceHandler creates a new ResourceHandler with injected dependencies.
func NewResourceHandler(store *model.ResourceStore, categoryStore *model.CategoryStore,
	dataDir string,
	sm *scs.SessionManager, userStore *model.UserStore, playlistStore *model.PlaylistStore,
	transcodeQueue *transcode.Queue) *ResourceHandler {
	return &ResourceHandler{
		store:          store,
		categoryStore:  categoryStore,
		playlistStore:  playlistStore,
		transcodeQueue: transcodeQueue,
		dataDir:        dataDir,
		sm:             sm,
		userStore:      userStore,
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
	password := r.FormValue("password")
	categoryID := r.FormValue("category_id")
	readme := r.FormValue("readme")

	// Category is required.
	if categoryID == "" {
		respondError(w, r, http.StatusBadRequest, "Category is required")
		return
	}

	userID := middleware.GetUserID(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)

	// Non-admin users: verify authorization for non-global categories.
	if userRole != "admin" && model.RequiresPassword(categoryID) {
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
	if model.IsPublic(categoryID) {
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

	// Save to temp file while computing BLAKE3 hash via TeeReader.
	tmpFile, err := os.CreateTemp(h.dataDir, "upload-*")
	if err != nil {
		slog.Error("failed to create temp file", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}
	defer os.Remove(tmpFile.Name())

	hashWriter := blake3.New(32, nil)
	teeReader := io.TeeReader(file, hashWriter)
	if _, err = io.Copy(tmpFile, teeReader); err != nil {
		slog.Error("failed to write temp file", "error", err)
		tmpFile.Close()
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}
	tmpFile.Close()

	hashHex := hex.EncodeToString(hashWriter.Sum(nil))

	// Check for duplicate content by hash.
	// Duplicate detection uses content BLAKE3 hash as the resource ID (PK). The row and files are removed on delete, freeing the hash for re-upload of identical content.
	existing, err := h.store.GetByID(hashHex)
	if err == nil && existing != nil {
		respondError(w, r, http.StatusConflict, "File already exists: "+existing.ID)
		return
	}

	// Create hash-based directory structure.
	hashDir := storage.HashPath(h.dataDir, hashHex)
	if err := os.MkdirAll(hashDir, 0o755); err != nil {
		slog.Error("failed to create hash directory", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}

	// Move temp file to final location (must be on same filesystem as dataDir).
	originalPath := storage.OriginalPath(h.dataDir, hashHex)
	if err := os.Rename(tmpFile.Name(), originalPath); err != nil {
		slog.Error("failed to move temp file", "error", err)
		os.RemoveAll(hashDir)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}

	// Write readme file if provided.
	if readme != "" {
		if err := os.WriteFile(storage.ReadmePath(h.dataDir, hashHex), []byte(readme), 0o644); err != nil {
			slog.Error("failed to write readme", "error", err)
			os.RemoveAll(hashDir)
			respondError(w, r, http.StatusInternalServerError, "Storage error")
			return
		}
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}

	resource := &model.Resource{
		ID:           hashHex,
		Title:        title,
		PasswordHash: string(hash),
		Filename:     header.Filename,
		FileSize:     header.Size,
		ContentType:  contentType,
		UploadedBy:   userID,
		CategoryID:   categoryID,
	}

	if err := h.store.Insert(resource); err != nil {
		slog.Error("failed to insert resource", "error", err)
		os.RemoveAll(hashDir)
		respondError(w, r, http.StatusInternalServerError, "Failed to save record")
		return
	}

	slog.Info("resource uploaded",
		"id", hashHex,
		"title", title,
		"size", header.Size,
	)

	// Submit transcode job (non-blocking).
	h.transcodeQueue.Submit(transcode.Job{
		ResourceID: hashHex,
		InputPath:  originalPath,
		OutputDir:  storage.HLSPath(h.dataDir, hashHex),
	})

	respondJSONOK(w, map[string]interface{}{
		"redirect": "/admin",
	})
}

// ListResourcesAPI returns all resources as JSON (for populating dropdowns).
// GET /api/resources
func (h *ResourceHandler) ListResourcesAPI(w http.ResponseWriter, r *http.Request) {
	userRole := middleware.GetUserRole(r.Context(), h.sm)
	userID := middleware.GetUserID(r.Context(), h.sm)

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

	var resources []*model.Resource
	var total int
	var err error
	if userRole == "admin" {
		resources, err = h.store.ListPaginated(limit, offset)
		if err == nil {
			total, err = h.store.Count()
		}
	} else {
		resources, err = h.store.ListByUploaderPaginated(userID, limit, offset)
		if err == nil {
			total, err = h.store.CountByUploader(userID)
		}
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
		"total":     total,
		"limit":     limit,
		"offset":    offset,
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

	// Per-session view guard (primary watch signal at GetResourceAPI).
	// Uses 24h SCS session lifetime; prevents double-count on repeated fetches.
	if !h.sm.GetBool(r.Context(), "viewed:"+id) {
		h.sm.Put(r.Context(), "viewed:"+id, true)
		go func() {
			if err := h.store.IncrementViews(id); err != nil {
				slog.Error("increment views failed", "id", id, "error", err)
			}
		}()
	}

	// Sanitize — never expose password hash.
	resource.PasswordHash = ""

	// Read readme file if it exists.
	readmeContent := ""
	readmePath := storage.ReadmePath(h.dataDir, resource.ID)
	if data, err := os.ReadFile(readmePath); err == nil {
		readmeContent = string(data)
	}

	respondJSONOK(w, map[string]interface{}{
		"id":               resource.ID,
		"title":            resource.Title,
		"readme":           readmeContent,
		"filename":         resource.Filename,
		"file_size":        resource.FileSize,
		"content_type":     resource.ContentType,
		"views":            resource.Views,
		"transcode_status": resource.TranscodeStatus,
		"created_at":       resource.CreatedAt,
		"updated_at":       resource.UpdatedAt,
		"uploaded_by":      resource.UploadedBy,
		"category_id":      resource.CategoryID,
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
	hashDir := filepath.Clean(storage.HashPath(h.dataDir, id))
	videoBase := filepath.Clean(filepath.Join(h.dataDir, "video"))
	if !strings.HasPrefix(hashDir, videoBase) {
		respondError(w, r, http.StatusBadRequest, "Invalid path")
		return
	}

	// Delete frees the BLAKE3 hash (PK) so identical content can be re-uploaded later.
	err = h.store.DeleteWithFile(id, func() error {
		if err := os.RemoveAll(hashDir); err != nil && !os.IsNotExist(err) {
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

// UpdateReadme updates the readme file for a resource.
// PUT /api/resources/{id}/readme
func (h *ResourceHandler) UpdateReadme(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify resource exists.
	_, err := h.store.GetByID(id)
	if err != nil {
		respondError(w, r, http.StatusNotFound, "Resource not found")
		return
	}

	// Parse JSON body.
	var body struct {
		Readme string `json:"readme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, r, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Write readme file.
	readmePath := storage.ReadmePath(h.dataDir, id)
	if err := os.MkdirAll(filepath.Dir(readmePath), 0o755); err != nil {
		slog.Error("failed to create readme directory", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}
	if err := os.WriteFile(readmePath, []byte(body.Readme), 0o644); err != nil {
		slog.Error("failed to write readme", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}

	respondJSONOK(w, map[string]interface{}{"ok": true})
}
