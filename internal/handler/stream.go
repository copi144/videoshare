package handler

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"videoshare/internal/model"
	"videoshare/internal/storage"
)

// ServeHLS serves HLS playlist and segment files.
// GET /api/video/{id}/hls/{path}
func (h *StreamHandler) ServeHLS(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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
// GET /api/video/{id}
func (h *StreamHandler) ServeVideo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Reconstruct paths using storage helpers.
	originalPath := storage.OriginalPath(h.dataDir, id)
	cleanPath := filepath.Clean(originalPath)
	hashDir := filepath.Clean(storage.HashPath(h.dataDir, id))

	// Path traversal prevention.
	if !strings.HasPrefix(cleanPath, hashDir) {
		slog.Error("path traversal attempt", "id", id, "resolved", cleanPath)
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, cleanPath)
}
