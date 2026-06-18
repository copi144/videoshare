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
	"videoshare/internal/web"
)

// NewRouter creates and configures the chi router with all route groups.
func NewRouter(sm *scs.SessionManager, templates fs.FS,
	resourceStore *model.ResourceStore, dataDir string, db *sql.DB,
	userStore *model.UserStore, categoryStore *model.CategoryStore, playlistStore *model.PlaylistStore) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(sm.LoadAndSave)
	r.Use(middleware.RateLimit(60, time.Minute))

	// Health check
	r.Get("/health", NewHealthHandler(db).ServeHealth)

	// Homepage — simple redirect
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin", http.StatusFound)
	})

	// Create handlers with dependency injection.
	userH := NewUserHandler(userStore, sm, templates)
	authH := NewAuthHandler(resourceStore, sm, templates)
	resourceH := NewResourceHandler(resourceStore, categoryStore, templates, dataDir, sm, userStore, playlistStore)
	streamH := NewStreamHandler(resourceStore, dataDir)
	playlistH := NewPlaylistHandler(playlistStore, resourceStore, categoryStore, sm, templates)
	categoryH := NewCategoryHandler(categoryStore, userStore, sm, templates)

	// User login/logout — public routes
	r.Get("/login", userH.ServeLoginPage)
	r.With(middleware.RateLimit(10, time.Minute)).Post("/login", userH.Login)

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
		r.Delete("/api/resource/{id}", resourceH.DeleteResourceAPI)
		r.Post("/logout", userH.Logout)

		// JSON API routes that require auth
		r.Get("/api/resources", resourceH.ListResourcesAPI)
		r.Post("/api/logout", userH.ServeLogoutAPI)

		// Category management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			r.Get("/admin/categories", categoryH.ServeCategoriesPage)
			r.Post("/admin/categories", categoryH.CreateCategory)
			r.Delete("/admin/categories/{id}/delete", categoryH.DeleteCategory)
			r.Post("/admin/categories/{id}/uploaders", categoryH.AssignUploaders)

			// JSON API routes for categories (admin only)
			r.Post("/api/categories", categoryH.CreateCategoryAPI)
			r.Delete("/api/categories/{id}", categoryH.DeleteCategoryAPI)
			r.Post("/api/categories/{id}/uploaders", categoryH.AssignUploadersAPI)
		})

		// Playlist management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			r.Get("/admin/playlists", playlistH.ServePlaylistsPage)
			r.Post("/admin/playlists", playlistH.CreatePlaylist)
			r.Delete("/admin/playlists/{id}/delete", playlistH.DeletePlaylist)
			r.Post("/admin/playlists/{id}/videos", playlistH.AddVideoToPlaylist)
			r.Post("/admin/playlists/{id}/videos/remove", playlistH.RemoveVideoFromPlaylist)

			// JSON API routes for playlists (admin only)
			r.Post("/api/playlists", playlistH.CreatePlaylistAPI)
			r.Delete("/api/playlists/{id}", playlistH.DeletePlaylistAPI)
			r.Post("/api/playlists/{id}/videos", playlistH.AddVideoAPI)
			r.Delete("/api/playlists/{id}/videos/{resourceId}", playlistH.RemoveVideoAPI)
		})

		// User management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			r.Get("/admin/users", userH.ServeUsersPage)
			r.Post("/admin/users", userH.CreateUser)

			// JSON API routes for users (admin only)
			r.Post("/api/users", userH.CreateUserAPI)
		})
	})

	// JSON API routes that are public
	r.Post("/api/login", userH.ServeLoginAPI)
	r.Post("/api/s/{id}/auth", authH.AuthenticateAPI)

	// Video streaming — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm)).Get("/api/video/{id}", streamH.ServeVideo)

	// Static file serving for embedded assets
	staticFS := web.Static()
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))).ServeHTTP(w, r)
	})

	return r
}
