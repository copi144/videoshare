package handler

import (
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"golang.org/x/crypto/bcrypt"

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

// Login authenticates a user with username/password.
// POST /login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Guard clause: both fields are required.
	if username == "" || password == "" {
		_ = parseAndRender(w, h.templates, "login.html", &TemplateData{
			Title: "Login — VideoShare",
			Error: "Username and password are required.",
		})
		return
	}

	user, err := h.userStore.GetByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = parseAndRender(w, h.templates, "login.html", &TemplateData{
				Title: "Login — VideoShare",
				Error: "Invalid username or password.",
			})
			return
		}
		slog.Error("failed to lookup user", "username", username, "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		_ = parseAndRender(w, h.templates, "login.html", &TemplateData{
			Title: "Login — VideoShare",
			Error: "Invalid username or password.",
		})
		return
	}

	middleware.SetUserSession(r.Context(), h.sm, user.ID, user.Role, user.Username)
	slog.Info("user logged in", "username", username, "role", user.Role)

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// Logout clears the user session.
// POST /logout
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	middleware.ClearUserSession(r.Context(), h.sm)
	slog.Info("user logged out")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
