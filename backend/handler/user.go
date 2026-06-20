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
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
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
		expiresAt := time.Now().UTC().Add(30 * time.Minute)
		if dbErr := model.SaveAPIToken(h.db, apiToken, user.ID, user.Role, user.Username, expiresAt); dbErr != nil {
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

// CreateUserAPI creates a new user and returns TOTP setup info as JSON.
// POST /api/users
func (h *UserHandler) CreateUserAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
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

	// Default role to "uploader" if not specified.
	if req.Role == "" {
		req.Role = "uploader"
	}

	// Validate role value.
	validRoles := map[string]bool{"admin": true, "uploader": true, "user": true}
	if !validRoles[req.Role] {
		respondJSONError(w, "Invalid role. Must be 'admin', 'uploader', or 'user'.", http.StatusBadRequest)
		return
	}

	// Only root admin (username="admin") can create admin users.
	if req.Role == "admin" {
		callerUsername := middleware.GetUsername(r.Context(), h.sm)
		if callerUsername != "admin" {
			respondJSONError(w, "Only the root admin can create admin users", http.StatusForbidden)
			return
		}
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
		ID:          req.Username,
		Username:    req.Username,
		TotpSecret:  key.Secret(),
		DisplayName: req.DisplayName,
		Role:        req.Role,
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

	slog.Info("user created via API", "username", req.Username, "role", req.Role)

	respondJSONOK(w, map[string]interface{}{
		"totp_secret": key.Secret(),
		"totp_uri":    key.URL(),
		"qr_image":    qrDataURI,
		"redirect":    "/admin/users?created=" + url.QueryEscape(req.Username),
	})
}

// ListUsersAPI returns a list of all users.
// GET /api/users
func (h *UserHandler) ListUsersAPI(w http.ResponseWriter, r *http.Request) {
	users, err := h.userStore.List()
	if err != nil {
		slog.Error("failed to list users", "error", err)
		respondJSONError(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	// Build response list without exposing TOTP secrets.
	type userResp struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		CreatedAt   string `json:"created_at"`
	}
	resp := make([]userResp, 0, len(users))
	for _, u := range users {
		resp = append(resp, userResp{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Role:        u.Role,
			CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	respondJSONOK(w, map[string]interface{}{
		"users": resp,
	})
}

// DeleteUserAPI deletes a user by ID.
// DELETE /api/users/{id}
func (h *UserHandler) DeleteUserAPI(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		respondJSONError(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	// Look up the target user.
	targetUser, err := h.userStore.GetByID(targetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "User not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to look up target user", "id", targetID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Prevent deletion of the root admin account.
	if targetUser.Username == "admin" {
		respondJSONError(w, "Cannot delete the root admin account", http.StatusForbidden)
		return
	}

	// Check permissions of the calling user.
	callerUsername := middleware.GetUsername(r.Context(), h.sm)
	callerRole := middleware.GetUserRole(r.Context(), h.sm)

	if callerRole != "admin" {
		respondJSONError(w, "Only admins can delete users", http.StatusForbidden)
		return
	}

	// If the target user is an admin, only the root admin can delete them.
	if targetUser.Role == "admin" && callerUsername != "admin" {
		respondJSONError(w, "Only the root admin can delete other admin users", http.StatusForbidden)
		return
	}

	if err := h.userStore.Delete(targetID); err != nil {
		slog.Error("failed to delete user", "id", targetID, "error", err)
		respondJSONError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	slog.Info("user deleted via API", "id", targetID, "username", targetUser.Username)
	respondJSONOK(w, nil)
}

// ResetTOTPAPI generates a new TOTP key for the specified user.
// POST /api/users/{id}/reset-totp
func (h *UserHandler) ResetTOTPAPI(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		respondJSONError(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	// Look up the target user.
	targetUser, err := h.userStore.GetByID(targetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "User not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to look up target user", "id", targetID, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Check permissions of the calling user.
	callerUsername := middleware.GetUsername(r.Context(), h.sm)
	callerRole := middleware.GetUserRole(r.Context(), h.sm)

	if callerRole != "admin" {
		respondJSONError(w, "Only admins can reset TOTP", http.StatusForbidden)
		return
	}

	// If the target user is an admin, only the root admin can reset their TOTP.
	if targetUser.Role == "admin" && callerUsername != "admin" {
		respondJSONError(w, "Only the root admin can reset TOTP for other admin users", http.StatusForbidden)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VideoShare",
		AccountName: targetUser.Username,
	})
	if err != nil {
		slog.Error("failed to generate TOTP key", "error", err)
		respondJSONError(w, "Failed to generate TOTP key", http.StatusInternalServerError)
		return
	}

	if err := h.userStore.UpdateTotpSecret(targetID, key.Secret()); err != nil {
		slog.Error("failed to update TOTP secret", "id", targetID, "error", err)
		respondJSONError(w, "Failed to reset TOTP", http.StatusInternalServerError)
		return
	}

	slog.Info("TOTP reset via API", "id", targetID, "username", targetUser.Username)

	respondJSONOK(w, map[string]interface{}{
		"totp_secret": key.Secret(),
		"totp_uri":    key.URL(),
	})
}
