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

	"github.com/alexedwards/scs/v2"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"videoshare/middleware"
	"videoshare/model"
)

// SessionHandler handles session creation (login, share auth, token auth).
type SessionHandler struct {
	userStore     *model.UserStore
	resourceStore *model.ResourceStore
	sm            *scs.SessionManager
	db            *sql.DB
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(userStore *model.UserStore, resourceStore *model.ResourceStore, sm *scs.SessionManager, db *sql.DB) *SessionHandler {
	return &SessionHandler{
		userStore:     userStore,
		resourceStore: resourceStore,
		sm:            sm,
		db:            db,
	}
}

type sessionRequest struct {
	Type       string `json:"type"` // "user", "share", "token"
	Username   string `json:"username,omitempty"`
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
	if req.Username == "" || req.TOTPCode == "" {
		respondJSONError(w, "Username and authenticator code are required.", http.StatusBadRequest)
		return
	}

	user, err := h.userStore.GetByUsername(req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "Invalid username or code.", http.StatusUnauthorized)
			return
		}
		slog.Error("failed to lookup user", "username", req.Username, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !totp.Validate(req.TOTPCode, user.TotpSecret) {
		respondJSONError(w, "Invalid username or code.", http.StatusUnauthorized)
		return
	}

	middleware.SetUserSession(r.Context(), h.sm, user.ID, user.Role, user.Username)
	slog.Info("user logged in via /api/session", "username", req.Username)

	// Generate API token for Bearer auth on subsequent API calls.
	apiToken := ""
	tokenBytes := make([]byte, 32)
	if _, randErr := rand.Read(tokenBytes); randErr == nil {
		tokenStr := hex.EncodeToString(tokenBytes)
		if dbErr := model.SaveAPIToken(h.db, tokenStr, user.ID, user.Role, user.Username); dbErr == nil {
			apiToken = tokenStr
		} else {
			slog.Error("failed to save API token", "error", dbErr)
		}
	}

	respondJSONOK(w, map[string]interface{}{
		"ok":        true,
		"api_token": apiToken,
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
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

	if model.IsPublic(resource.CategoryID) {
		// Public/global category — auto-auth
		tokenBefore := h.sm.Token(r.Context())
		slog.Debug("handleShareSession before SetVideoAuth", "token", tokenBefore, "resourceID", req.ResourceID)

		middleware.SetVideoAuth(r.Context(), h.sm)

		// If request has a Bearer token from a logged-in user, bind user data to session too
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			tok := strings.TrimPrefix(auth, "Bearer ")
			if apiToken, err := model.GetAPIToken(h.db, tok); err == nil {
				middleware.SetUserSession(r.Context(), h.sm, apiToken.UserID, apiToken.UserRole, apiToken.Username)
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

	if req.Password == "" {
		respondJSONError(w, "Password is required.", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(resource.PasswordHash), []byte(req.Password)); err != nil {
		respondJSONError(w, "Invalid password.", http.StatusUnauthorized)
		return
	}

	tokenBefore := h.sm.Token(r.Context())
	slog.Debug("handleShareSession before SetVideoAuth", "token", tokenBefore, "resourceID", req.ResourceID, "hasPassword", true)

	middleware.SetVideoAuth(r.Context(), h.sm)

	// If request has a Bearer token from a logged-in user, bind user data to session too
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		tok := strings.TrimPrefix(auth, "Bearer ")
		if apiToken, err := model.GetAPIToken(h.db, tok); err == nil {
			middleware.SetUserSession(r.Context(), h.sm, apiToken.UserID, apiToken.UserRole, apiToken.Username)
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

	middleware.SetUserSession(r.Context(), h.sm, apiToken.UserID, apiToken.UserRole, apiToken.Username)
	slog.Info("session created from token", "user", apiToken.Username)

	respondJSONOK(w, map[string]interface{}{
		"ok": true,
		"user": map[string]string{
			"id":       apiToken.UserID,
			"username": apiToken.Username,
			"role":     apiToken.UserRole,
		},
	})
}
