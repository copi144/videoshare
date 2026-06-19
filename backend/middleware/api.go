package middleware

import (
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
)

// APIAuth validates API requests via Authorization: Bearer token.
// The token must match the api_token stored in the session (set on login).
// This prevents CSRF attacks by requiring a header that cross-origin requests cannot set.
// Only applies to /api/ paths (not /v/, /health, or SPA routes).
func APIAuth(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only for /api/ paths
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Public API endpoints don't need API token.
			// /api/login — public login endpoint
			// /api/me — returns current user info; needed during page rehydration to bootstrap api_token
			// /api/s/ — share-link authentication
			if r.URL.Path == "/api/login" || r.URL.Path == "/api/me" || strings.HasPrefix(r.URL.Path, "/api/s/") {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, `{"error":"Missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")
			storedToken := sm.GetString(r.Context(), "api_token")

			if storedToken == "" || storedToken != token {
				http.Error(w, `{"error":"Invalid authorization token"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
