package middleware

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
)

const (
	sessionAuthenticatedKey = "authenticated"
	sessionUserIDKey        = "user_id"
	sessionUserRoleKey      = "user_role"
	sessionUsernameKey      = "user_username"
)

// RequireVideoAuth returns middleware that protects video watch routes behind
// a valid video-password-authenticated session. If the "authenticated" key is
// not present in the session, the user is redirected to the share page.
func RequireVideoAuth(sm *scs.SessionManager) func(http.Handler) http.Handler {
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

// RequireUserOrVideoAuth checks for EITHER system user auth OR video viewer auth.
// This allows both admin/uploader users and share-link viewers to access the same route.
func RequireUserOrVideoAuth(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := sm.GetString(r.Context(), sessionUserIDKey)
			videoAuth := sm.GetBool(r.Context(), sessionAuthenticatedKey)
			if userID == "" && !videoAuth {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SetVideoAuth marks the current session as authenticated for video viewing.
func SetVideoAuth(ctx context.Context, sm *scs.SessionManager) {
	sm.Put(ctx, sessionAuthenticatedKey, true)
}

// SetUserSession stores the authenticated user's ID, role, and username in the session.
func SetUserSession(ctx context.Context, sm *scs.SessionManager, userID, role, username string) {
	sm.Put(ctx, sessionUserIDKey, userID)
	sm.Put(ctx, sessionUserRoleKey, role)
	sm.Put(ctx, sessionUsernameKey, username)
}

// ClearUserSession removes user authentication data from the session.
func ClearUserSession(ctx context.Context, sm *scs.SessionManager) {
	sm.Remove(ctx, sessionUserIDKey)
	sm.Remove(ctx, sessionUserRoleKey)
	sm.Remove(ctx, sessionUsernameKey)
}

// GetUserID returns the authenticated user's ID from the session.
// Returns empty string if not set.
func GetUserID(ctx context.Context, sm *scs.SessionManager) string {
	return sm.GetString(ctx, sessionUserIDKey)
}

// GetUserRole returns the authenticated user's role from the session.
// Returns empty string if not set.
func GetUserRole(ctx context.Context, sm *scs.SessionManager) string {
	return sm.GetString(ctx, sessionUserRoleKey)
}

// GetUsername returns the authenticated user's username from the session.
// Returns empty string if not set.
func GetUsername(ctx context.Context, sm *scs.SessionManager) string {
	return sm.GetString(ctx, sessionUsernameKey)
}

// RequireUserAuth returns middleware that protects routes behind a valid
// user authentication session. If no user_id is found in the session,
// the user is redirected to the login page.
func RequireUserAuth(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetUserID(r.Context(), sm) == "" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin returns middleware that restricts access to users with the
// "admin" role. Returns 403 Forbidden if the user is not an admin.
func RequireAdmin(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetUserRole(r.Context(), sm) != "admin" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
