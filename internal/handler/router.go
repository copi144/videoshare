package handler

import (
	"database/sql"
	"io/fs"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"videoshare/internal/middleware"
	"videoshare/internal/model"
)

// NewRouter creates and configures the chi router with all route groups.
// csrfKey is a 32-byte hex-encoded key; csrfSecure controls the Secure flag on the CSRF cookie.
func NewRouter(sm *scs.SessionManager, templates fs.FS, csrfKey []byte, csrfSecure bool,
	resourceStore *model.ResourceStore, dataDir string, db *sql.DB,
	userStore *model.UserStore, categoryStore *model.CategoryStore, playlistStore *model.PlaylistStore) http.Handler {
	r := chi.NewRouter()

	// Health check — before any middleware (no auth, no CSRF required)
	r.Get("/health", NewHealthHandler(db).ServeHealth)

	// Homepage — simple redirect
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin", http.StatusFound)
	})

	// Global middleware
	// Method override: support _method form field for HTML forms
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				if m := r.FormValue("_method"); m != "" {
					r.Method = m
				}
			}
			next.ServeHTTP(w, r)
		})
	})
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(sm.LoadAndSave)
	r.Use(middleware.CSRFProtection(csrfKey, csrfSecure))
	r.Use(middleware.RateLimit(60, time.Minute))

	// Create handlers with dependency injection.
	userH := NewUserHandler(userStore, sm, templates)
	authH := NewAuthHandler(resourceStore, sm, templates)
	resourceH := NewResourceHandler(resourceStore, categoryStore, templates, dataDir, sm, userStore, playlistStore)
	streamH := NewStreamHandler(resourceStore, dataDir)
	playlistH := NewPlaylistHandler(playlistStore, resourceStore, categoryStore, sm, templates)

	// User login/logout — public routes
	r.Get("/login", userH.ServeLoginPage)
	r.Post("/login", userH.Login)

	// Public routes — password entry for shared videos (tighter rate limit)
	r.Route("/s/{id}", func(r chi.Router) {
		r.Use(middleware.RateLimit(5, time.Minute))

		r.Get("/", authH.ServeSharePage)
		r.Post("/auth", authH.Authenticate)

		// Watch page requires video session authentication
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireVideoAuth(sm))
			r.Get("/watch", authH.ServeWatchPage)
		})
	})

	// Admin area — requires user authentication (login)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireUserAuth(sm))

		r.Get("/admin", resourceH.List)
		r.Post("/api/upload", resourceH.Upload)
		r.Delete("/api/resource/{id}", resourceH.Delete)
		r.Post("/logout", userH.Logout)

		// Category management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			categoryH := NewCategoryHandler(categoryStore, userStore, sm, templates)
			r.Get("/admin/categories", categoryH.ServeCategoriesPage)
			r.Post("/admin/categories", categoryH.CreateCategory)
			r.Delete("/admin/categories/{id}/delete", categoryH.DeleteCategory)
			r.Post("/admin/categories/{id}/uploaders", categoryH.AssignUploaders)
		})

		// Playlist management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			r.Get("/admin/playlists", playlistH.ServePlaylistsPage)
			r.Post("/admin/playlists", playlistH.CreatePlaylist)
			r.Delete("/admin/playlists/{id}/delete", playlistH.DeletePlaylist)
			r.Post("/admin/playlists/{id}/videos", playlistH.AddVideoToPlaylist)
			r.Post("/admin/playlists/{id}/videos/remove", playlistH.RemoveVideoFromPlaylist)
		})
	})

	// Video streaming — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm)).Get("/api/video/{id}", streamH.ServeVideo)

	// Static file serving for embedded assets
	// TODO: serve embedded static files from web directory

	return r
}
