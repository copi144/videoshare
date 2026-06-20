package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/pquerna/otp/totp"

	"videoshare/middleware"
	"videoshare/model"
)

// SessionHandler handles session creation (login, share auth, token auth).
type SessionHandler struct {
	userStore          *model.UserStore
	resourceStore      *model.ResourceStore
	shareResourceStore *model.ShareResourceStore
	categoryStore      *model.CategoryStore
	sm                 *scs.SessionManager
	db                 *sql.DB
	shareLinkStore     *model.ShareLinkStore
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(userStore *model.UserStore, resourceStore *model.ResourceStore, shareResourceStore *model.ShareResourceStore, categoryStore *model.CategoryStore, sm *scs.SessionManager, db *sql.DB, shareLinkStore *model.ShareLinkStore) *SessionHandler {
	return &SessionHandler{
		userStore:          userStore,
		resourceStore:      resourceStore,
		shareResourceStore: shareResourceStore,
		categoryStore:      categoryStore,
		sm:                 sm,
		db:                 db,
		shareLinkStore:     shareLinkStore,
	}
}

type sessionRequest struct {
	Type       string `json:"type"` // "user", "share", "token"
	Name       string `json:"name,omitempty"`
	TOTPCode   string `json:"totp_code,omitempty"`
	ResourceID string `json:"resource_id,omitempty"`
	Password   string `json:"password,omitempty"`
	Token      string `json:"token,omitempty"`
}

// ServeSessionAPI creates or updates a session.
// POST /api/session
func (h *SessionHandler) ServeSessionAPI(w http.ResponseWriter, r *http.Request) {
	var req sessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	switch req.Type {
	case "user":
		h.handleUserSession(w, r, req)
	case "share":
		h.handleShareSession(w, r, req)
	case "token":
		h.handleTokenSession(w, r, req)
	default:
		respondJSONError(w, "Invalid session type. Use 'user', 'share', or 'token'.", http.StatusBadRequest)
	}
}

func (h *SessionHandler) handleUserSession(w http.ResponseWriter, r *http.Request, req sessionRequest) {
	if req.Name == "" || req.TOTPCode == "" {
		respondJSONError(w, "Name and authenticator code are required.", http.StatusBadRequest)
		return
	}

	user, err := h.userStore.GetByName(req.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "Invalid name or code.", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to lookup user", "name", req.Name, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !totp.Validate(req.TOTPCode, user.TotpSecret) {
		respondJSONError(w, "Invalid name or code.", http.StatusUnauthorized)
		return
	}

	middleware.SetUserSession(r.Context(), h.sm, user.Name, user.IsAdmin)
	slog.Info("user logged in via /api/session", "name", req.Name)

	// Generate API token for Bearer auth on subsequent API calls.
	apiToken := ""
	tokenBytes := make([]byte, 32)
	if _, randErr := rand.Read(tokenBytes); randErr == nil {
		tokenStr := hex.EncodeToString(tokenBytes)
		expiresAt := time.Now().UTC().Add(30 * time.Minute)
		if dbErr := model.SaveAPIToken(h.db, tokenStr, user.Name, expiresAt); dbErr == nil {
			apiToken = tokenStr
		} else {
			slog.Error("failed to save API token", "error", dbErr)
		}
	}

	respondJSONOK(w, map[string]interface{}{
		"ok":        true,
		"api_token": apiToken,
		"user": map[string]interface{}{
			"name":     user.Name,
			"is_admin": user.IsAdmin,
		},
	})
}

func (h *SessionHandler) handleShareSession(w http.ResponseWriter, r *http.Request, req sessionRequest) {
	if req.ResourceID == "" {
		respondJSONError(w, "Resource ID is required.", http.StatusBadRequest)
		return
	}

	resource, err := h.resourceStore.GetByID(req.ResourceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "Resource not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to load resource", "id", req.ResourceID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if resource.Banned {
		respondJSONError(w, "This video has been banned", http.StatusGone)
		return
	}

	// If session already has share link scope (from category/playlist share link auth),
	// check if the requested resource is within scope
	targetType := h.sm.GetString(r.Context(), "share_target_type")
	targetID := h.sm.GetString(r.Context(), "share_target_id")
	if targetType != "" && targetID != "" && req.Password == "" {
		if h.shareLinkStore.IsResourceInScope(req.ResourceID, targetType, targetID) {
			slog.Debug("share link session: resource in scope, auto-auth",
				"resource_id", req.ResourceID, "target_type", targetType, "target_id", targetID)
			middleware.SetVideoAuth(r.Context(), h.sm)
			respondJSONOK(w, map[string]interface{}{"ok": true, "redirect": "/#/v/" + req.ResourceID + "/watch"})
			return
		}
	}

	// Load resource categories to determine access.
	resourceCats, _ := h.resourceStore.GetResourceCategories(resource.ID)
	isPublic := false
	for _, c := range resourceCats {
		if model.IsPublic(c) {
			isPublic = true
			break
		}
	}

	// If user is authenticated and has category access, auto-auth
	userID := middleware.GetUserID(r.Context(), h.sm)
	if userID != "" && !isPublic {
		isAdmin := middleware.GetIsAdmin(r.Context(), h.sm)
		// Admin can access everything
		if isAdmin {
			middleware.SetVideoAuth(r.Context(), h.sm)
			respondJSONOK(w, map[string]interface{}{"ok": true, "redirect": "/#/v/" + req.ResourceID + "/watch"})
			return
		}
		// Check if user is assigned to any non-public category of this resource
		for _, c := range resourceCats {
			if !model.IsPublic(c) {
				assigned, err := h.categoryStore.IsAssigned(userID, c)
				if err == nil && assigned {
					middleware.SetVideoAuth(r.Context(), h.sm)
					respondJSONOK(w, map[string]interface{}{"ok": true, "redirect": "/#/v/" + req.ResourceID + "/watch"})
					return
				}
			}
		}
	}

	if isPublic {
		// Public/global category — auto-auth
		tokenBefore := h.sm.Token(r.Context())
		slog.Debug("handleShareSession before SetVideoAuth", "token", tokenBefore, "resourceID", req.ResourceID)

		middleware.SetVideoAuth(r.Context(), h.sm)

		// If request has a Bearer token from a logged-in user, bind user data to session too
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			tok := strings.TrimPrefix(auth, "Bearer ")
			if apiToken, err := model.GetAPIToken(h.db, tok); err == nil {
				isAdmin := apiToken.IsAdmin
				middleware.SetUserSession(r.Context(), h.sm, apiToken.Name, isAdmin)
			}
		}

		tokenAfter := h.sm.Token(r.Context())
		slog.Debug("handleShareSession after auth", "token", tokenAfter, "hasUserID", h.sm.GetString(r.Context(), "user_id") != "", "authenticated", h.sm.GetBool(r.Context(), "authenticated"))

		respondJSONOK(w, map[string]interface{}{
			"ok":       true,
			"redirect": "/#/v/" + req.ResourceID + "/watch",
		})
		return
	}

	// If password is empty and user has no access, tell them they need a share link
	if req.Password == "" {
		respondJSONError(w, "This video requires a share link to access.", http.StatusUnauthorized)
		return
	}

	// Validate against share_resources table
	if _, err := h.shareResourceStore.GetByResourceAndPassword(req.ResourceID, req.Password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "Invalid or expired link.", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to validate share link", "resource_id", req.ResourceID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tokenBefore := h.sm.Token(r.Context())
	slog.Debug("handleShareSession before SetVideoAuth", "token", tokenBefore, "resourceID", req.ResourceID, "hasPassword", true)

	middleware.SetVideoAuth(r.Context(), h.sm)

	// If request has a Bearer token from a logged-in user, bind user data to session too
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		tok := strings.TrimPrefix(auth, "Bearer ")
		if apiToken, err := model.GetAPIToken(h.db, tok); err == nil {
			isAdmin := apiToken.IsAdmin
			middleware.SetUserSession(r.Context(), h.sm, apiToken.Name, isAdmin)
		}
	}

	tokenAfter := h.sm.Token(r.Context())
	slog.Debug("handleShareSession after auth", "token", tokenAfter, "hasUserID", h.sm.GetString(r.Context(), "user_id") != "", "authenticated", h.sm.GetBool(r.Context(), "authenticated"))

	respondJSONOK(w, map[string]interface{}{
		"ok":       true,
		"redirect": "/#/v/" + req.ResourceID + "/watch",
	})
}

func (h *SessionHandler) handleTokenSession(w http.ResponseWriter, r *http.Request, req sessionRequest) {
	tokenStr := req.Token
	if tokenStr == "" {
		// Fallback to Authorization header
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if tokenStr == "" {
		respondJSONError(w, "Token is required.", http.StatusBadRequest)
		return
	}

	apiToken, err := model.GetAPIToken(h.db, tokenStr)
	if err != nil {
		respondJSONError(w, "Invalid token.", http.StatusUnauthorized)
		return
	}

	isAdmin := apiToken.IsAdmin
	middleware.SetUserSession(r.Context(), h.sm, apiToken.Name, isAdmin)
	slog.Info("session created from token", "user", apiToken.Name)

	respondJSONOK(w, map[string]interface{}{
		"ok": true,
		"user": map[string]interface{}{
			"name":     apiToken.Name,
			"is_admin": isAdmin,
		},
	})
}
