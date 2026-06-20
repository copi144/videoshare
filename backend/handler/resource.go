package handler

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"lukechampine.com/blake3"

	"videoshare/middleware"
	"videoshare/model"
	"videoshare/storage"
	"videoshare/transcode"
	"videoshare/upload"
)

var viewedResource sync.Map

// ResourceHandler handles CRUD operations for video resources.
type ResourceHandler struct {
	store           *model.ResourceStore
	categoryStore   *model.CategoryStore
	playlistStore   *model.PlaylistStore
	transcodeQueue  *transcode.Queue
	dataDir         string
	userStore       *model.UserStore
	ffmpegPath      string
}

// NewResourceHandler creates a new ResourceHandler with injected dependencies.
func NewResourceHandler(store *model.ResourceStore, categoryStore *model.CategoryStore,
	dataDir string, userStore *model.UserStore, playlistStore *model.PlaylistStore,
	transcodeQueue *transcode.Queue, ffmpegPath string) *ResourceHandler {
	return &ResourceHandler{
		store:          store,
		categoryStore:  categoryStore,
		playlistStore:  playlistStore,
		transcodeQueue: transcodeQueue,
		dataDir:        dataDir,
		userStore:      userStore,
		ffmpegPath:     ffmpegPath,
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
	categoryName := r.FormValue("category_id")
	readme := r.FormValue("readme")
	noTranscode := r.FormValue("no_transcode") == "1"

	// Category is required.
	if categoryName == "" {
		respondError(w, r, http.StatusBadRequest, "Category is required")
		return
	}
	if title == "" {
		respondError(w, r, http.StatusBadRequest, "Title is required")
		return
	}

	userID := middleware.GetUserIDFromContext(r.Context())
	isAdmin := middleware.GetIsAdminFromContext(r.Context())

	// Non-admin users: verify authorization for non-global categories.
	if !isAdmin && !model.IsPublic(categoryName) {
		authorized, authErr := h.categoryStore.CanUpload(userID, categoryName)
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

	// Detect resource type from magic bytes (never use file extension).
	contentType, resourceType, detectErr := upload.DetectMIMEAndResourceType(file)
	if detectErr != nil || resourceType == "" {
		slog.Error("file type detection failed", "error", detectErr, "filename", header.Filename)
		respondJSONError(w, "Unable to detect file type. Supported formats: MP4, WebM, MKV, MOV, AVI, MP3, M4A, WAV, OGG, FLAC, AAC, JPEG, PNG, WebP, GIF", http.StatusBadRequest)
		return
	}

	// Auto-correct filename extension to match detected MIME type.
	correctedFilename := upload.CorrectExtension(header.Filename, contentType)
	if correctedFilename != header.Filename {
		slog.Info("auto-corrected filename extension",
			"original", header.Filename,
			"corrected", correctedFilename,
			"detected_mime", contentType,
		)
		header.Filename = correctedFilename
	}

	// Save to temp file while computing BLAKE3 hash via TeeReader.
	tmpDir := filepath.Join(h.dataDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		slog.Error("failed to create temp directory", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}
	tmpFile, err := os.CreateTemp(tmpDir, "upload-*")
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

	// Video-specific validation (dimensions, duration).
	if resourceType == storage.ResourceTypeVideo {
		dims, probeErr := upload.ProbeVideoDimensions(upload.FFprobePath(h.ffmpegPath), tmpFile.Name())
		if probeErr != nil {
			slog.Warn("failed to probe video dimensions, skipping validation", "error", probeErr)
		} else {
			// Validate: short side must be >= 144 pixels
			if dims.MinSide() < upload.MinShortSide {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				respondError(w, r, http.StatusBadRequest,
					fmt.Sprintf("Video too small: short side is %dpx, minimum is %dpx", dims.MinSide(), upload.MinShortSide))
				return
			}
			// Validate: aspect ratio must not exceed 4:1
			if dims.AspectRatio() > upload.MaxAspectRatio {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				respondError(w, r, http.StatusBadRequest,
					fmt.Sprintf("Video aspect ratio %.1f:1 exceeds maximum 4:1", dims.AspectRatio()))
				return
			}
			// Auto-disable transcoding for videos with short side < 360px
			if dims.MinSide() < upload.NoTranscodeSide {
				noTranscode = true
			}
		}

		// Probe video duration.
		duration, durErr := upload.ProbeVideoDuration(upload.FFprobePath(h.ffmpegPath), tmpFile.Name())
		if durErr != nil {
			slog.Warn("failed to probe video duration, skipping validation", "error", durErr)
		} else if duration < upload.MinDuration {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			respondError(w, r, http.StatusBadRequest,
				fmt.Sprintf("Video too short: %.2f seconds, minimum is %.0f second", duration, upload.MinDuration))
			return
		}
	}

	// Check for duplicate content by hash.
	existing, err := h.store.GetByID(hashHex)
	if err == nil && existing != nil {
		if existing.Banned {
			respondError(w, r, http.StatusConflict, "This file has been banned and cannot be re-uploaded")
			return
		}

		// File exists: check if already linked to this category.
		cats, catErr := h.store.GetResourceCategories(hashHex)
		if catErr == nil {
			for _, c := range cats {
				if c == categoryName {
					respondError(w, r, http.StatusConflict, "File already exists in this category: "+hashHex)
					return
				}
			}
		}

		// Not in this category: create a reference link only.
		if err := h.store.AddResourceCategory(hashHex, categoryName); err != nil {
			slog.Error("failed to add resource category for existing file", "id", hashHex, "category", categoryName, "error", err)
			respondError(w, r, http.StatusInternalServerError, "Failed to reference existing file")
			return
		}

		slog.Info("resource linked to new category (dedup)", "id", hashHex, "category", categoryName)
		respondJSONOK(w, map[string]interface{}{
			"redirect": "/admin",
			"linked":   true,
		})
		return
	}

	// Create hash-based directory structure.
	hashDir := storage.HashPath(h.dataDir, resourceType, hashHex)
	if err := os.MkdirAll(hashDir, 0o755); err != nil {
		slog.Error("failed to create hash directory", "error", err)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}

	// Move temp file to final location (must be on same filesystem as dataDir).
	originalPath := storage.OriginalPath(h.dataDir, resourceType, hashHex)
	if err := os.Rename(tmpFile.Name(), originalPath); err != nil {
		slog.Error("failed to move temp file", "error", err)
		os.RemoveAll(hashDir)
		respondError(w, r, http.StatusInternalServerError, "Storage error")
		return
	}

	// Write readme file if provided.
	if readme != "" {
		if err := os.WriteFile(storage.ReadmePath(h.dataDir, resourceType, hashHex), []byte(readme), 0o644); err != nil {
			slog.Error("failed to write readme", "error", err)
			os.RemoveAll(hashDir)
			respondError(w, r, http.StatusInternalServerError, "Storage error")
			return
		}
	}

	resource := &model.Resource{
		ID:           hashHex,
		Title:        title,
		Filename:     header.Filename,
		FileSize:     header.Size,
		ContentType:  contentType,
		ResourceType: resourceType,
		UploadedBy:   userID,
		NoTranscode:  noTranscode,
	}

	if err := h.store.Insert(resource); err != nil {
		slog.Error("failed to insert resource", "error", err)
		os.RemoveAll(hashDir)
		respondError(w, r, http.StatusInternalServerError, "Failed to save record")
		return
	}

	// Add resource to the selected category.
	if err := h.store.AddResourceCategory(hashHex, categoryName); err != nil {
		slog.Error("failed to add resource category, upload will proceed but resource may not appear in category filters",
			"id", hashHex, "category", categoryName, "error", err)
	}

	slog.Info("resource uploaded",
		"id", hashHex,
		"title", title,
		"size", header.Size,
	)

	// Submit transcode job (non-blocking), unless skipped.
	if resourceType == storage.ResourceTypeVideo {
		if !noTranscode {
			h.transcodeQueue.Submit(transcode.Job{
				ResourceID: hashHex,
				InputPath:  originalPath,
				OutputDir:  storage.HLSPath(h.dataDir, hashHex),
			})
		} else {
			slog.Info("transcode skipped by uploader request", "resource_id", hashHex)
		}
	} else {
		// Audio and image resources always get processed (no noTranscode flag).
		h.transcodeQueue.Submit(transcode.Job{
			ResourceID: hashHex,
			InputPath:  originalPath,
			OutputDir:  storage.HashPath(h.dataDir, resourceType, hashHex),
		})
	}

	respondJSONOK(w, map[string]interface{}{
		"redirect": "/admin",
	})
}

// ListResourcesAPI returns all resources as JSON (for populating dropdowns).
// GET /api/resources
func (h *ResourceHandler) ListResourcesAPI(w http.ResponseWriter, r *http.Request) {
	isAdmin := middleware.GetIsAdminFromContext(r.Context())
	userID := middleware.GetUserIDFromContext(r.Context())

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
	resourceType := r.URL.Query().Get("resource_type")
	categoryName := r.URL.Query().Get("category_name")

	switch {
	case categoryName != "":
		if isAdmin {
			resources, err = h.store.ListByCategoryPaginated(categoryName, limit, offset)
			if err == nil {
				total, err = h.store.CountByCategory(categoryName)
			}
		} else {
			resources, err = h.store.ListByCategoryAndUploaderPaginated(categoryName, userID, limit, offset)
			if err == nil {
				total, err = h.store.CountByCategoryAndUploader(categoryName, userID)
			}
		}
	case resourceType != "":
		if isAdmin {
			resources, err = h.store.ListByTypePaginated(resourceType, limit, offset)
			if err == nil {
				total, err = h.store.CountByType(resourceType)
			}
		} else {
			resources, err = h.store.ListByTypeAndUploaderPaginated(resourceType, userID, limit, offset)
			if err == nil {
				total, err = h.store.CountByTypeAndUploader(resourceType, userID)
			}
		}
	case isAdmin:
		resources, err = h.store.ListPaginated(limit, offset)
		if err == nil {
			total, err = h.store.Count()
		}
	default:
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

	// Filter banned resources for non-admin users
	if !isAdmin {
		filtered := make([]*model.Resource, 0, len(resources))
		for _, res := range resources {
			if !res.Banned {
				filtered = append(filtered, res)
			}
		}
		resources = filtered
	}

	// Enrich resources with their categories from the join table.
	if len(resources) > 0 {
		if err := h.store.EnrichWithCategories(resources); err != nil {
			slog.Error("failed to enrich resources with categories", "error", err)
		}
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

	// In-memory view guard to prevent double-count on repeated fetches.
	if _, loaded := viewedResource.LoadOrStore(id, true); !loaded {
		go func() {
			if err := h.store.IncrementViews(id); err != nil {
				slog.Error("increment views failed", "id", id, "error", err)
			}
		}()
	}

	// Populate categories from the join table.
	if cats, err := h.store.GetResourceCategories(resource.ID); err == nil {
		resource.Categories = cats
	}

	// Read readme file if it exists.
	readmeContent := ""
	readmePath := storage.ReadmePath(h.dataDir, resource.ResourceType, resource.ID)
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
		"resource_type":    resource.ResourceType,
		"views":            resource.Views,
		"banned":           resource.Banned,
		"transcode_status": resource.TranscodeStatus,
		"created_at":       resource.CreatedAt,
		"updated_at":       resource.UpdatedAt,
		"uploaded_by":      resource.UploadedBy,
		"categories":       resource.Categories,
	})
}

// DeleteResourceAPI removes a video resource and its file, or unlinks from a category.
// DELETE /api/resource/{id}
// Query params:
//
//	category_name: if set and resource is in multiple categories, just unlink from this category
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
	userID := middleware.GetUserIDFromContext(r.Context())
	isAdmin := middleware.GetIsAdminFromContext(r.Context())
	if !isAdmin && resource.UploadedBy != userID {
		respondError(w, r, http.StatusForbidden, "You can only delete your own videos")
		return
	}

	categoryName := r.URL.Query().Get("category_name")

	// If category_name is provided, check how many categories this resource is in.
	if categoryName != "" {
		catCount, catErr := h.store.GetResourceCategoriesCount(id)
		if catErr != nil {
			slog.Error("failed to count resource categories", "id", id, "error", catErr)
			respondError(w, r, http.StatusInternalServerError, "Failed to check category count")
			return
		}

		// If more than 1 category, just unlink from this one (keep the file).
		if catCount > 1 {
			if err := h.store.RemoveResourceFromCategory(id, categoryName); err != nil {
				slog.Error("failed to remove resource from category", "id", id, "category", categoryName, "error", err)
				respondError(w, r, http.StatusInternalServerError, "Failed to remove from category")
				return
			}
			slog.Info("resource unlinked from category", "id", id, "category", categoryName)
			respondJSONOK(w, map[string]interface{}{"unlinked": true, "file_deleted": false})
			return
		}
		// catCount <= 1: fall through to full delete below.
	}

	// Path traversal prevention.
	hashDir := filepath.Clean(storage.HashPath(h.dataDir, resource.ResourceType, resource.ID))
	typeBase := filepath.Clean(filepath.Join(h.dataDir, resource.ResourceType))
	if !strings.HasPrefix(hashDir, typeBase) {
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
	respondJSONOK(w, map[string]interface{}{"file_deleted": true})
}

// UpdateReadme updates the readme file for a resource.
// PUT /api/resources/{id}/readme
func (h *ResourceHandler) UpdateReadme(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify resource exists.
	res, err := h.store.GetByID(id)
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
	readmePath := storage.ReadmePath(h.dataDir, res.ResourceType, id)
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

// Retranscode triggers re-transcoding of a video.
// POST /api/resources/{id}/retranscode
func (h *ResourceHandler) Retranscode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.store.GetByID(id)
	if err != nil {
		respondError(w, r, http.StatusNotFound, "Resource not found")
		return
	}

	if resource.Banned {
		respondError(w, r, http.StatusGone, "This resource has been banned")
		return
	}

	// Check ownership
	userID := middleware.GetUserIDFromContext(r.Context())
	isAdmin := middleware.GetIsAdminFromContext(r.Context())
	if !isAdmin && resource.UploadedBy != userID {
		respondError(w, r, http.StatusForbidden, "You can only retranscode your own videos")
		return
	}

	// Check if transcode is already in progress
	if resource.TranscodeStatus == "processing" {
		respondError(w, r, http.StatusConflict, "Transcode already in progress")
		return
	}

	// Set status to pending and submit job
	if err := h.store.UpdateTranscodeStatus(id, "pending"); err != nil {
		slog.Error("failed to update transcode status", "id", id, "error", err)
		respondError(w, r, http.StatusInternalServerError, "Failed to start transcode")
		return
	}

	h.transcodeQueue.Submit(transcode.Job{
		ResourceID: id,
		InputPath:  storage.OriginalPath(h.dataDir, resource.ResourceType, id),
		OutputDir:  storage.HLSPath(h.dataDir, id),
	})

	slog.Info("retranscode triggered", "id", id)
	respondJSONOK(w, map[string]interface{}{"ok": true})
}

// BanResource bans a video: deletes video data from disk, prevents re-upload.
// POST /api/resources/{id}/ban
func (h *ResourceHandler) BanResource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Admin only check
	isAdmin := middleware.GetIsAdminFromContext(r.Context())
	if !isAdmin {
		respondError(w, r, http.StatusForbidden, "Admin access required")
		return
	}

	resource, err := h.store.GetByID(id)
	if err != nil {
		respondError(w, r, http.StatusNotFound, "Resource not found")
		return
	}

	if resource.Banned {
		respondError(w, r, http.StatusConflict, "Video is already banned")
		return
	}

	// Delete video data from disk (original file + HLS output). Preserve readme file.

	// Remove original video file
	originalPath := storage.OriginalPath(h.dataDir, resource.ResourceType, resource.ID)
	if err := os.Remove(originalPath); err != nil && !os.IsNotExist(err) {
		slog.Error("failed to remove original file during ban", "id", id, "error", err)
	}

	// Remove HLS output (video only)
	if resource.ResourceType == storage.ResourceTypeVideo {
		hlsDir := storage.HLSPath(h.dataDir, resource.ID)
		if err := os.RemoveAll(hlsDir); err != nil && !os.IsNotExist(err) {
			slog.Error("failed to remove HLS dir during ban", "id", id, "error", err)
		}
	}

	// Set banned flag in DB (preserves metadata + readme on disk)
	if err := h.store.SetBanned(id, true); err != nil {
		slog.Error("failed to set banned flag", "id", id, "error", err)
		respondError(w, r, http.StatusInternalServerError, "Failed to ban video")
		return
	}

	// Update transcode status to 'none' since data is gone
	if err := h.store.UpdateTranscodeStatus(id, "none"); err != nil {
		slog.Error("failed to update transcode status after ban", "id", id, "error", err)
	}

	slog.Info("resource banned", "id", id, "title", resource.Title)
	respondJSONOK(w, map[string]interface{}{"ok": true})
}
