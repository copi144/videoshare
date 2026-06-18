package middleware

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
)

const sessionAuthenticatedKey = "authenticated"

// SessionAuth returns middleware that protects routes behind a valid session.
// If the "authenticated" key is not present in the session, the user is
// redirected to the login page.
func SessionAuth(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := sm.GetBool(r.Context(), sessionAuthenticatedKey)
			if !auth {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SetAuthenticated marks the current session as authenticated.
func SetAuthenticated(ctx context.Context, sm *scs.SessionManager) {
	sm.Put(ctx, sessionAuthenticatedKey, true)
}
