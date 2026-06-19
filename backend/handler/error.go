package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

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
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// respondJSONError writes a JSON error response
func respondJSONError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// respondJSONOK writes a success JSON response
func respondJSONOK(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if data == nil {
		data = map[string]interface{}{"ok": true}
	} else {
		data["ok"] = true
	}
	json.NewEncoder(w).Encode(data)
}
