package middleware

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"

	"videoshare/model"
)

// APIAuth validates API requests via Authorization: Bearer token.
// The token must match a stored api_token in the api_tokens database table.
// This decouples API authentication from the session cookie, allowing
// cookie-free access (e.g., from SPA localStorage or programmatic clients).
// Only applies to /api/ paths (not /v/, /health, or SPA routes).
func APIAuth(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Public API endpoints don't need API token.
			// /api/login — public login endpoint
			// /api/s/ — share-link authentication
			if r.URL.Path == "/api/login" || strings.HasPrefix(r.URL.Path, "/api/s/") {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, `{"error":"Missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")
			apiToken, err := model.GetAPIToken(db, token)
			if err != nil {
				slog.Warn("invalid API token", "error", err)
				http.Error(w, `{"error":"Invalid authorization token"}`, http.StatusUnauthorized)
				return
			}

			// Set user info in context so downstream handlers can access it
			// without relying on the session cookie.
			ctx := SetUserContext(r.Context(), apiToken.UserID, apiToken.UserRole, apiToken.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
