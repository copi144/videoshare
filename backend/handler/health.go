package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// HealthHandler provides health check endpoints.
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// ServeHealth returns the application health status.
// GET /health
func (h *HealthHandler) ServeHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"

	if err := h.db.Ping(); err != nil {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  status,
		"service": "videoshare",
	})
}
