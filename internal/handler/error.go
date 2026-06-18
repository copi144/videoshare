package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorResponse represents a structured JSON error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// respondError sends a JSON error response with status code and logs it.
func respondError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	slog.Error("HTTP error",
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"error", msg,
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: http.StatusText(status), Message: msg})
}

// respondJSON sends a JSON success response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
