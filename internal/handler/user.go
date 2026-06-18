package handler

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image/png"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"videoshare/internal/middleware"
	"videoshare/internal/model"
)

// UserHandler handles system user authentication (login/logout) for the admin area.
type UserHandler struct {
	userStore *model.UserStore
	sm        *scs.SessionManager
	templates fs.FS
}

// NewUserHandler creates a new UserHandler with injected dependencies.
func NewUserHandler(userStore *model.UserStore, sm *scs.SessionManager, templates fs.FS) *UserHandler {
	return &UserHandler{userStore: userStore, sm: sm, templates: templates}
}

// ServeLoginPage renders the login form.
// GET /login
func (h *UserHandler) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to /admin.
	if middleware.GetUserID(r.Context(), h.sm) != "" {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	if err := parseAndRender(w, h.templates, "login.html", &TemplateData{
		Title: "Login — VideoShare",
	}); err != nil {
		slog.Error("failed to render login template", "error", err)
	}
}

// Login authenticates a user with username + TOTP code.
// POST /login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	code := r.FormValue("totp_code")

	// Guard clause: both fields are required.
	if username == "" || code == "" {
		_ = parseAndRender(w, h.templates, "login.html", &TemplateData{
			Title: "Login — VideoShare",
			Error: "Username and authenticator code are required.",
		})
		return
	}

	user, err := h.userStore.GetByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = parseAndRender(w, h.templates, "login.html", &TemplateData{
				Title: "Login — VideoShare",
				Error: "Invalid username or code.",
			})
			return
		}
		slog.Error("failed to lookup user", "username", username, "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if !totp.Validate(code, user.TotpSecret) {
		_ = parseAndRender(w, h.templates, "login.html", &TemplateData{
			Title: "Login — VideoShare",
			Error: "Invalid username or code.",
		})
		return
	}

	middleware.SetUserSession(r.Context(), h.sm, user.ID, user.Role, user.Username)
	slog.Info("user logged in", "username", username, "role", user.Role)

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// ServeUsersPage lists all users and allows admin to create new uploaders.
// GET /admin/users
func (h *UserHandler) ServeUsersPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	username := middleware.GetUsername(r.Context(), h.sm)
	userRole := middleware.GetUserRole(r.Context(), h.sm)

	users, err := h.userStore.List()
	if err != nil {
		slog.Error("failed to list users", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	errorMsg := r.URL.Query().Get("error")

	data := map[string]interface{}{
		"Users": users,
	}

	// If ?created= param is set, read TOTP info from session and clear immediately.
	if createdUsername := r.URL.Query().Get("created"); createdUsername != "" {
		if secret := h.sm.GetString(r.Context(), "created_user_secret"); secret != "" {
			createdUser := map[string]string{
				"Username":  h.sm.GetString(r.Context(), "created_user_name"),
				"Secret":    secret,
				"TOTPURI":   h.sm.GetString(r.Context(), "created_user_uri"),
				"QRDataURI": h.sm.GetString(r.Context(), "created_user_qr"),
			}
			data["CreatedUser"] = createdUser

			// Clear session data to prevent showing TOTP info again.
			h.sm.Remove(r.Context(), "created_user_name")
			h.sm.Remove(r.Context(), "created_user_secret")
			h.sm.Remove(r.Context(), "created_user_uri")
			h.sm.Remove(r.Context(), "created_user_qr")
		}
	}

	if err := parseAndRender(w, h.templates, "users.html", &TemplateData{
		Title:      "Users — VideoShare",
		IsLoggedIn: true,
		Username:   username,
		UserRole:   userRole,
		Error:      errorMsg,
		Data:       data,
	}); err != nil {
		slog.Error("failed to render users template", "error", err)
	}
}

// CreateUser creates a new uploader user with a TOTP key and displays the setup info.
// POST /admin/users
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")

	// Guard clause: username is required.
	if username == "" {
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Username is required"), http.StatusSeeOther)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VideoShare",
		AccountName: username,
	})
	if err != nil {
		slog.Error("failed to generate TOTP key", "error", err)
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Failed to create user"), http.StatusSeeOther)
		return
	}

	user := &model.User{
		ID:         uuid.New().String(),
		Username:   username,
		TotpSecret: key.Secret(),
		Role:       "uploader",
	}

	if err := h.userStore.Insert(user); err != nil {
		slog.Error("failed to insert user", "username", username, "error", err)
		http.Redirect(w, r, "/admin/users?error="+url.QueryEscape("Failed to create user"), http.StatusSeeOther)
		return
	}

	// Store TOTP info in session and redirect so a refresh won't lose the secret.
	h.sm.Put(r.Context(), "created_user_name", username)
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

	http.Redirect(w, r, "/admin/users?created="+url.QueryEscape(username), http.StatusSeeOther)
}

// Logout clears the user session.
// POST /logout
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearUserSession(r.Context(), h.sm)
	slog.Info("user logged out")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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
		ID:         uuid.New().String(),
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
