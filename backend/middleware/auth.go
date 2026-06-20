package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"

	"videoshare/model"
)

const (
	sessionAuthenticatedKey = "authenticated"
	sessionUserIDKey        = "user_id"
	sessionIsAdminKey       = "is_admin"
	sessionUsernameKey      = "user_name"
)

type ctxKey string

const (
	ctxUserID   ctxKey = "ctx_user_id"
	ctxIsAdmin  ctxKey = "ctx_is_admin"
	ctxUsername         ctxKey = "ctx_username"
	ctxAPIAuthenticated ctxKey = "ctx_api_authenticated"
)

// SetUserContext returns a new context with user identity values set.
// This is used by APIAuth to propagate user info without relying on the session.
func SetUserContext(ctx context.Context, name string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, ctxUserID, name)
	ctx = context.WithValue(ctx, ctxIsAdmin, isAdmin)
	ctx = context.WithValue(ctx, ctxUsername, name)
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
				slog.Debug("RequireUserOrVideoAuth redirecting", "path", r.URL.Path, "userID", userID, "videoAuth", videoAuth, "sessionUserID", sm.GetString(r.Context(), "user_id"), "sessionAuth", sm.GetBool(r.Context(), "authenticated"), "token", sm.Token(r.Context()))
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

// SetUserSession stores the authenticated user's name and admin status in the session.
func SetUserSession(ctx context.Context, sm *scs.SessionManager, name string, isAdmin bool) {
	sm.Put(ctx, sessionUserIDKey, name)
	sm.Put(ctx, sessionIsAdminKey, isAdmin)
	sm.Put(ctx, sessionUsernameKey, name)
}

// ClearUserSession removes user authentication data from the session.
func ClearUserSession(ctx context.Context, sm *scs.SessionManager) {
	sm.Remove(ctx, sessionUserIDKey)
	sm.Remove(ctx, sessionIsAdminKey)
	sm.Remove(ctx, sessionUsernameKey)
}

// GetUserIDFromContext returns the user name from the context alone, without
// needing a session manager. This is safe when the caller knows that APIAuth
// or similar middleware has already populated the context.
func GetUserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(ctxUserID).(string); ok && id != "" {
		return id
	}
	return ""
}

// GetIsAdminFromContext returns the user's admin status from the context alone.
func GetIsAdminFromContext(ctx context.Context) bool {
	if isAdmin, ok := ctx.Value(ctxIsAdmin).(bool); ok {
		return isAdmin
	}
	return false
}

// GetUserID returns the authenticated user's name, checking context first, then session.
// Returns empty string if not set.
func GetUserID(ctx context.Context, sm *scs.SessionManager) string {
	if id, ok := ctx.Value(ctxUserID).(string); ok && id != "" {
		return id
	}
	return sm.GetString(ctx, sessionUserIDKey)
}

// GetIsAdmin returns whether the authenticated user is an admin.
func GetIsAdmin(ctx context.Context, sm *scs.SessionManager) bool {
	if isAdmin, ok := ctx.Value(ctxIsAdmin).(bool); ok {
		return isAdmin
	}
	return sm.GetBool(ctx, sessionIsAdminKey)
}

// GetUsername returns the authenticated user's name, checking context first, then session.
// Returns empty string if not set.
func GetUsername(ctx context.Context, sm *scs.SessionManager) string {
	if name, ok := ctx.Value(ctxUsername).(string); ok && name != "" {
		return name
	}
	return sm.GetString(ctx, sessionUsernameKey)
}

// SetAPIAuthenticated returns a context marked as API-token-authenticated.
func SetAPIAuthenticated(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxAPIAuthenticated, true)
}

// IsAPIAuthenticated returns true if the request was authenticated via API token.
func IsAPIAuthenticated(ctx context.Context) bool {
	if v, ok := ctx.Value(ctxAPIAuthenticated).(bool); ok {
		return v
	}
	return false
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

// RequireAdmin returns middleware that restricts access to admin users.
// Returns 403 Forbidden if the user is not an admin.
func RequireAdmin(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !GetIsAdmin(r.Context(), sm) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireShareScope returns middleware that restricts resource access to what is within
// the share link scope (for session-authenticated, non-user requests).
// Logged-in users bypass scope checks entirely.
func RequireShareScope(sm *scs.SessionManager, store *model.ShareLinkStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Logged-in users bypass scope
			if GetUserID(r.Context(), sm) != "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if session has share scope
			targetType := sm.GetString(r.Context(), "share_target_type")
			targetID := sm.GetString(r.Context(), "share_target_id")
			if targetType == "" || targetID == "" {
				// No scope = blanket access (per-resource share visitor)
				next.ServeHTTP(w, r)
				return
			}

			// Extract resource ID from URL
			resourceID := chi.URLParam(r, "id")
			if resourceID == "" {
				slog.Warn("RequireShareScope: no resource ID in URL", "path", r.URL.Path)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Check cache first (authorized_resources map in session)
			authMap, ok := sm.Get(r.Context(), "authorized_resources").(map[string]bool)
			if ok && authMap[resourceID] {
				next.ServeHTTP(w, r)
				return
			}

			// Check scope via database
			if !store.IsResourceInScope(resourceID, targetType, targetID) {
				slog.Warn("RequireShareScope: resource not in scope",
					"resource_id", resourceID, "target_type", targetType, "target_id", targetID)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Cache the result
			if authMap == nil {
				authMap = make(map[string]bool)
			}
			authMap[resourceID] = true
			sm.Put(r.Context(), "authorized_resources", authMap)

			next.ServeHTTP(w, r)
		})
	}
}
