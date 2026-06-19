package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"videoshare/model"
	"videoshare/storage"
)

// ServeHLS serves HLS playlist and segment files.
// GET /v/{id}/hls/{path}
func (h *StreamHandler) ServeHLS(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check if resource is banned
	resource, err := h.store.GetByID(id)
	if err != nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	if resource.Banned {
		http.Error(w, "This video has been banned", http.StatusGone)
		return
	}

	// Get the wildcard path after /hls/.
	wildcard := chi.RouteContext(r.Context()).URLParam("*")
	if wildcard == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	// Construct the file path.
	hlsDir := storage.HLSPath(h.dataDir, id)
	filePath := filepath.Join(hlsDir, wildcard)
	cleanPath := filepath.Clean(filePath)

	// Path traversal prevention.
	if !strings.HasPrefix(cleanPath, filepath.Clean(hlsDir)) {
		slog.Error("path traversal attempt in HLS", "id", id, "path", wildcard, "resolved", cleanPath)
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Set correct content type for m3u8 files.
	if strings.HasSuffix(wildcard, ".m3u8") {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	} else if strings.HasSuffix(wildcard, ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
	}

	http.ServeFile(w, r, cleanPath)
}

// StreamHandler handles video streaming with range request support.
type StreamHandler struct {
	store   *model.ResourceStore
	dataDir string
}

// NewStreamHandler creates a new StreamHandler with injected dependencies.
func NewStreamHandler(store *model.ResourceStore, dataDir string) *StreamHandler {
	return &StreamHandler{store: store, dataDir: dataDir}
}

// ServeVideo streams a video file with HTTP range request support.
// GET /v/{id}
func (h *StreamHandler) ServeVideo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check if resource is banned
	resource, err := h.store.GetByID(id)
	if err != nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	if resource.ResourceType != storage.ResourceTypeVideo {
		http.Error(w, "Not a video resource", http.StatusBadRequest)
		return
	}
	if resource.Banned {
		http.Error(w, "This video has been banned", http.StatusGone)
		return
	}

	// Reconstruct paths using storage helpers.
	originalPath := storage.OriginalPath(h.dataDir, storage.ResourceTypeVideo, id)
	cleanPath := filepath.Clean(originalPath)
	hashDir := filepath.Clean(storage.HashPath(h.dataDir, storage.ResourceTypeVideo, id))

	// Path traversal prevention.
	if !strings.HasPrefix(cleanPath, hashDir) {
		slog.Error("path traversal attempt", "id", id, "resolved", cleanPath)
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// If video is transcoded (done), don't serve the original file.
	// The video is only available via HLS.
	if resource.TranscodeStatus == "done" && !resource.NoTranscode {
		http.Error(w, "Video only available via HLS streaming", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, cleanPath)
}

// ServeDownload serves the original video file for download, regardless of transcode status.
// GET /v/{id}/download
func (h *StreamHandler) ServeDownload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.store.GetByID(id)
	if err != nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	if resource.ResourceType != storage.ResourceTypeVideo {
		http.Error(w, "Not a video resource", http.StatusBadRequest)
		return
	}

	if resource.Banned {
		http.Error(w, "This video has been banned", http.StatusGone)
		return
	}

	originalPath := storage.OriginalPath(h.dataDir, storage.ResourceTypeVideo, id)
	cleanPath := filepath.Clean(originalPath)
	hashDir := filepath.Clean(storage.HashPath(h.dataDir, storage.ResourceTypeVideo, id))

	if !strings.HasPrefix(cleanPath, hashDir) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Set Content-Disposition to force download with original filename.
	filename := resource.Filename
	if filename == "" {
		filename = id + ".mp4"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Type", resource.ContentType)

	http.ServeFile(w, r, cleanPath)
}

// ServeAudio streams an audio file.
// GET /a/{id}
func (h *StreamHandler) ServeAudio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.store.GetByID(id)
	if err != nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	if resource.ResourceType != storage.ResourceTypeAudio {
		http.Error(w, "Not an audio resource", http.StatusBadRequest)
		return
	}
	if resource.Banned {
		http.Error(w, "This resource has been banned", http.StatusGone)
		return
	}

	audioPath := storage.AudioOutputPath(h.dataDir, id)
	cleanPath := filepath.Clean(audioPath)
	hashDir := filepath.Clean(storage.HashPath(h.dataDir, storage.ResourceTypeAudio, id))
	if !strings.HasPrefix(cleanPath, hashDir) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, cleanPath)
}

// ServeImage serves an image file, preferring the downsampled variant.
// GET /i/{id}
func (h *StreamHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resource, err := h.store.GetByID(id)
	if err != nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	if resource.ResourceType != storage.ResourceTypeImage {
		http.Error(w, "Not an image resource", http.StatusBadRequest)
		return
	}
	if resource.Banned {
		http.Error(w, "This resource has been banned", http.StatusGone)
		return
	}

	imagePath := storage.OriginalPath(h.dataDir, storage.ResourceTypeImage, id)
	// First try the processed image (downsampled).
	processed := storage.ImagePath(h.dataDir, id, "thumb.jpg")
	cleanProcessed := filepath.Clean(processed)
	hashDir := filepath.Clean(storage.HashPath(h.dataDir, storage.ResourceTypeImage, id))
	if strings.HasPrefix(cleanProcessed, hashDir) {
		if _, err := os.Stat(cleanProcessed); err == nil {
			http.ServeFile(w, r, cleanProcessed)
		return
		}
	}
	// Fall back to original.
	originalPath := filepath.Clean(imagePath)
	if !strings.HasPrefix(originalPath, hashDir) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, originalPath)
}
