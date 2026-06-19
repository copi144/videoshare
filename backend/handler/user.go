package handler

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image/png"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/pquerna/otp/totp"

	"videoshare/middleware"
	"videoshare/model"
)

// UserHandler handles system user authentication (login/logout) for the admin area.
type UserHandler struct {
	userStore *model.UserStore
	sm        *scs.SessionManager
	db        *sql.DB
}

// NewUserHandler creates a new UserHandler with injected dependencies.
func NewUserHandler(userStore *model.UserStore, sm *scs.SessionManager, db *sql.DB) *UserHandler {
	return &UserHandler{userStore: userStore, sm: sm, db: db}
}

// ServeLoginAPI handles JSON login requests.
// POST /api/login
func (h *UserHandler) ServeLoginAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		TOTPCode string `json:"totp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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
	slog.Info("user logged in", "username", req.Username, "role", user.Role)

	// Generate API token for Bearer auth on subsequent API calls.
	apiTokenBytes := make([]byte, 32)
	if _, randErr := rand.Read(apiTokenBytes); randErr == nil {
		apiToken := hex.EncodeToString(apiTokenBytes)

		// Store in api_tokens table for cookie-free API auth
		if dbErr := model.SaveAPIToken(h.db, apiToken, user.ID, user.Role, user.Username); dbErr != nil {
			slog.Error("failed to save API token", "error", dbErr)
		}

		respondJSONOK(w, map[string]interface{}{
			"ok":        true,
			"redirect":  "/admin",
			"api_token": apiToken,
		})
		return
	}

	respondJSONOK(w, map[string]interface{}{
		"ok":       true,
		"redirect": "/admin",
	})
}

// ServeLogoutAPI handles JSON logout requests.
// POST /api/logout
func (h *UserHandler) ServeLogoutAPI(w http.ResponseWriter, r *http.Request) {
	// Delete API token from DB
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		if err := model.DeleteAPIToken(h.db, token); err != nil {
			slog.Error("failed to delete API token", "error", err)
		}
	}

	middleware.ClearUserSession(r.Context(), h.sm)
	h.sm.Remove(r.Context(), "api_token")
	slog.Info("user logged out via API")
	respondJSONOK(w, map[string]interface{}{
		"redirect": "/login",
	})
}

// ServeMeAPI returns the current authenticated user's info as JSON.
// GET /api/me
func (h *UserHandler) ServeMeAPI(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context(), h.sm)
	username := middleware.GetUsername(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)

	if userID == "" {
		respondJSONOK(w, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	respondJSONOK(w, map[string]interface{}{
		"authenticated": true,
		"user": map[string]string{
			"id":       userID,
			"username": username,
			"role":     userRole,
		},
	})
}

// ServeHeartbeat refreshes the session idle timeout.
// POST /api/heartbeat
func (h *UserHandler) ServeHeartbeat(w http.ResponseWriter, r *http.Request) {
	respondJSONOK(w, map[string]interface{}{"ok": true})
}

// CreateUserAPI creates a new uploader user and returns TOTP setup info as JSON.
// POST /api/users
func (h *UserHandler) CreateUserAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		respondJSONError(w, "Username is required", http.StatusBadRequest)
		return
	}

	if !model.IsValidName(req.Username) {
		respondJSONError(w, "Username must only contain letters, numbers, and hyphens", http.StatusBadRequest)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VideoShare",
		AccountName: req.Username,
	})
	if err != nil {
		slog.Error("failed to generate TOTP key", "error", err)
		respondJSONError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	user := &model.User{
		ID:         req.Username,
		Username:   req.Username,
		TotpSecret: key.Secret(),
		Role:       "uploader",
	}

	if err := h.userStore.Insert(user); err != nil {
		slog.Error("failed to insert user", "username", req.Username, "error", err)
		respondJSONError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Store TOTP info in session for the redirect page to pick up.
	h.sm.Put(r.Context(), "created_user_name", req.Username)
	h.sm.Put(r.Context(), "created_user_secret", key.Secret())
	h.sm.Put(r.Context(), "created_user_uri", key.URL())

	// Generate QR code as a base64 data URI.
	qrDataURI := ""
	img, err := key.Image(256, 256)
	if err != nil {
		slog.Warn("failed to generate QR image", "error", err)
	} else {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			slog.Warn("failed to encode QR image as PNG", "error", err)
		} else {
			qrDataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
		}
	}
	h.sm.Put(r.Context(), "created_user_qr", qrDataURI)

	slog.Info("user created via API", "username", req.Username)

	respondJSONOK(w, map[string]interface{}{
		"totp_secret": key.Secret(),
		"totp_uri":    key.URL(),
		"qr_image":    qrDataURI,
		"redirect":    "/admin/users?created=" + url.QueryEscape(req.Username),
	})
}
