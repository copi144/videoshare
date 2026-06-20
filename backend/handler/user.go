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
		Name    string `json:"name"`
		TOTPCode string `json:"totp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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
	slog.Info("user logged in", "name", req.Name, "is_admin", user.IsAdmin)

	// Generate API token for Bearer auth on subsequent API calls.
	apiTokenBytes := make([]byte, 32)
	if _, randErr := rand.Read(apiTokenBytes); randErr == nil {
		apiToken := hex.EncodeToString(apiTokenBytes)

		// Store in api_tokens table for cookie-free API auth
		expiresAt := time.Now().UTC().Add(30 * time.Minute)
		if dbErr := model.SaveAPIToken(h.db, apiToken, user.Name, expiresAt); dbErr != nil {
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
	isAdmin := middleware.GetIsAdmin(r.Context(), h.sm)

	if userID == "" {
		respondJSONOK(w, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	respondJSONOK(w, map[string]interface{}{
		"authenticated": true,
		"user": map[string]interface{}{
			"name":     username,
			"is_admin": isAdmin,
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
		Name        string `json:"name"`
		IsAdmin     bool   `json:"is_admin"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondJSONError(w, "Name is required", http.StatusBadRequest)
		return
	}

	if !model.IsValidName(req.Name) {
		respondJSONError(w, "Name must only contain letters, numbers, and hyphens", http.StatusBadRequest)
		return
	}

	// Only root admin (name="admin") can create admin users.
	if req.IsAdmin {
		callerName := middleware.GetUsername(r.Context(), h.sm)
		if callerName != "admin" {
			respondJSONError(w, "Only the root admin can create admin users", http.StatusForbidden)
			return
		}
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VideoShare",
		AccountName: req.Name,
	})
	if err != nil {
		slog.Error("failed to generate TOTP key", "error", err)
		respondJSONError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	user := &model.User{
		Name:        req.Name,
		TotpSecret:  key.Secret(),
		DisplayName: req.DisplayName,
		IsAdmin:     req.IsAdmin,
	}

	if err := h.userStore.Insert(user); err != nil {
		slog.Error("failed to insert user", "name", req.Name, "error", err)
		respondJSONError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Store TOTP info in session for the redirect page to pick up.
	h.sm.Put(r.Context(), "created_user_name", req.Name)
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

	slog.Info("user created via API", "name", req.Name, "is_admin", req.IsAdmin)

	respondJSONOK(w, map[string]interface{}{
		"totp_secret": key.Secret(),
		"totp_uri":    key.URL(),
		"qr_image":    qrDataURI,
		"redirect":    "/admin/users?created=" + url.QueryEscape(req.Name),
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
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		IsAdmin     bool   `json:"is_admin"`
		CreatedAt   string `json:"created_at"`
	}
	resp := make([]userResp, 0, len(users))
	for _, u := range users {
		resp = append(resp, userResp{
			Name:        u.Name,
			DisplayName: u.DisplayName,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	respondJSONOK(w, map[string]interface{}{
		"users": resp,
	})
}

// DeleteUserAPI deletes a user by name.
// DELETE /api/users/{name}
func (h *UserHandler) DeleteUserAPI(w http.ResponseWriter, r *http.Request) {
	targetName := chi.URLParam(r, "name")
	if targetName == "" {
		respondJSONError(w, "Missing name", http.StatusBadRequest)
		return
	}

	// Look up the target user.
	targetUser, err := h.userStore.GetByName(targetName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "User not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to look up target user", "name", targetName, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Prevent deletion of the root admin account.
	if targetUser.Name == "admin" {
		respondJSONError(w, "Cannot delete the root admin account", http.StatusForbidden)
		return
	}

	// Check permissions of the calling user.
	callerName := middleware.GetUsername(r.Context(), h.sm)
	callerIsAdmin := middleware.GetIsAdmin(r.Context(), h.sm)

	if !callerIsAdmin {
		respondJSONError(w, "Only admins can delete users", http.StatusForbidden)
		return
	}

	// If the target user is an admin, only the root admin can delete them.
	if targetUser.IsAdmin && callerName != "admin" {
		respondJSONError(w, "Only the root admin can delete other admin users", http.StatusForbidden)
		return
	}

	if err := h.userStore.Delete(targetName); err != nil {
		slog.Error("failed to delete user", "name", targetName, "error", err)
		respondJSONError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	slog.Info("user deleted via API", "name", targetName)
	respondJSONOK(w, nil)
}

// ResetTOTPAPI generates a new TOTP key for the specified user.
// POST /api/users/{name}/reset-totp
func (h *UserHandler) ResetTOTPAPI(w http.ResponseWriter, r *http.Request) {
	targetName := chi.URLParam(r, "name")
	if targetName == "" {
		respondJSONError(w, "Missing name", http.StatusBadRequest)
		return
	}

	// Look up the target user.
	targetUser, err := h.userStore.GetByName(targetName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondJSONError(w, "User not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to look up target user", "name", targetName, "error", err)
		respondJSONError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Check permissions of the calling user.
	callerName := middleware.GetUsername(r.Context(), h.sm)
	callerIsAdmin := middleware.GetIsAdmin(r.Context(), h.sm)

	if !callerIsAdmin {
		respondJSONError(w, "Only admins can reset TOTP", http.StatusForbidden)
		return
	}

	// If the target user is an admin, only the root admin can reset their TOTP.
	if targetUser.IsAdmin && callerName != "admin" {
		respondJSONError(w, "Only the root admin can reset TOTP for other admin users", http.StatusForbidden)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VideoShare",
		AccountName: targetUser.Name,
	})
	if err != nil {
		slog.Error("failed to generate TOTP key", "error", err)
		respondJSONError(w, "Failed to generate TOTP key", http.StatusInternalServerError)
		return
	}

	if err := h.userStore.UpdateTotpSecret(targetName, key.Secret()); err != nil {
		slog.Error("failed to update TOTP secret", "name", targetName, "error", err)
		respondJSONError(w, "Failed to reset TOTP", http.StatusInternalServerError)
		return
	}

	slog.Info("TOTP reset via API", "name", targetName)

	respondJSONOK(w, map[string]interface{}{
		"totp_secret": key.Secret(),
		"totp_uri":    key.URL(),
	})
}
