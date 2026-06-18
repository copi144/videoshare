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
