package handler

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image/png"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/alexedwards/scs/v2"
	"github.com/pquerna/otp/totp"

	"videoshare/internal/middleware"
	"videoshare/internal/model"
)

// UserHandler handles system user authentication (login/logout) for the admin area.
type UserHandler struct {
	userStore *model.UserStore
	sm        *scs.SessionManager
}

// NewUserHandler creates a new UserHandler with injected dependencies.
func NewUserHandler(userStore *model.UserStore, sm *scs.SessionManager) *UserHandler {
	return &UserHandler{userStore: userStore, sm: sm}
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

	respondJSONOK(w, map[string]interface{}{
		"redirect": "/admin",
	})
}

// ServeLogoutAPI handles JSON logout requests.
// POST /api/logout
func (h *UserHandler) ServeLogoutAPI(w http.ResponseWriter, r *http.Request) {
	middleware.ClearUserSession(r.Context(), h.sm)
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
