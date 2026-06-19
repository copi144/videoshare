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

type ctxKey string

const (
	ctxUserID   ctxKey = "ctx_user_id"
	ctxUserRole ctxKey = "ctx_user_role"
	ctxUsername ctxKey = "ctx_username"
)

// SetUserContext returns a new context with user identity values set.
// This is used by APIAuth to propagate user info without relying on the session.
func SetUserContext(ctx context.Context, userID, role, username string) context.Context {
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxUserRole, role)
	ctx = context.WithValue(ctx, ctxUsername, username)
	return ctx
}

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
			userID := GetUserID(r.Context(), sm)
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

// GetUserID returns the authenticated user's ID, checking context first, then session.
// Returns empty string if not set.
func GetUserID(ctx context.Context, sm *scs.SessionManager) string {
	if id, ok := ctx.Value(ctxUserID).(string); ok && id != "" {
		return id
	}
	return sm.GetString(ctx, sessionUserIDKey)
}

// GetUserRole returns the authenticated user's role, checking context first, then session.
// Returns empty string if not set.
func GetUserRole(ctx context.Context, sm *scs.SessionManager) string {
	if role, ok := ctx.Value(ctxUserRole).(string); ok && role != "" {
		return role
	}
	return sm.GetString(ctx, sessionUserRoleKey)
}

// GetUsername returns the authenticated user's username, checking context first, then session.
// Returns empty string if not set.
func GetUsername(ctx context.Context, sm *scs.SessionManager) string {
	if name, ok := ctx.Value(ctxUsername).(string); ok && name != "" {
		return name
	}
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
