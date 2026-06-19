package handler

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"videoshare/middleware"
	"videoshare/model"
	"videoshare/transcode"
	"videoshare/web"
)

// NewRouter creates and configures the chi router with all route groups.
func NewRouter(sm *scs.SessionManager,
	resourceStore *model.ResourceStore, dataDir string, db *sql.DB,
	userStore *model.UserStore, categoryStore *model.CategoryStore, playlistStore *model.PlaylistStore,
	transcodeQueue *transcode.Queue, ffmpegPath string) (http.Handler, []func()) {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(sm.LoadAndSave)

	rl, stop := middleware.RateLimit(60, time.Minute)
	r.Use(rl)

	r.Use(middleware.APIAuth(sm))

	var stops []func()
	stops = append(stops, stop)

	// Health check
	r.Get("/health", NewHealthHandler(db).ServeHealth)

	// Homepage — serve the SPA
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		spa, err := web.SPA()
		if err != nil {
			slog.Error("failed to read SPA", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(spa)
	})

	// Create handlers with dependency injection.
	userH := NewUserHandler(userStore, sm)
	authH := NewAuthHandler(resourceStore, sm)
	resourceH := NewResourceHandler(resourceStore, categoryStore, dataDir, sm, userStore, playlistStore, transcodeQueue, ffmpegPath)
	streamH := NewStreamHandler(resourceStore, dataDir)
	playlistH := NewPlaylistHandler(playlistStore, resourceStore, categoryStore, sm)
	categoryH := NewCategoryHandler(categoryStore, userStore, sm)

	// JSON API routes that are public
	r.Post("/api/login", userH.ServeLoginAPI)
	r.Post("/api/s/{id}/auth", authH.AuthenticateAPI)

	// Admin area — requires user authentication (login)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireUserAuth(sm))

		r.Post("/api/upload", resourceH.Upload)
		r.Put("/api/resources/{id}/readme", resourceH.UpdateReadme)
		r.Delete("/api/resource/{id}", resourceH.DeleteResourceAPI)
		r.Post("/api/resources/{id}/retranscode", resourceH.Retranscode)
		r.Post("/api/resources/{id}/ban", resourceH.BanResource)

		// JSON API routes that require auth
		r.Get("/api/resources", resourceH.ListResourcesAPI)
		r.Get("/api/me", userH.ServeMeAPI)
		r.Get("/api/categories", categoryH.ListCategoriesAPI)
		r.Post("/api/logout", userH.ServeLogoutAPI)
		r.Post("/api/heartbeat", userH.ServeHeartbeat)

		// Category management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			// JSON API routes for categories (admin only)
			r.Post("/api/categories", categoryH.CreateCategoryAPI)
			r.Delete("/api/categories/{id}", categoryH.DeleteCategoryAPI)
			r.Post("/api/categories/{id}/uploaders", categoryH.AssignUploadersAPI)
		})

		// Playlist management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			// JSON API routes for playlists (admin only)
			r.Get("/api/playlists", playlistH.ListPlaylistsAPI)
			r.Post("/api/playlists", playlistH.CreatePlaylistAPI)
			r.Delete("/api/playlists/{id}", playlistH.DeletePlaylistAPI)
			r.Post("/api/playlists/{id}/videos", playlistH.AddVideoAPI)
			r.Delete("/api/playlists/{id}/videos/{resourceId}", playlistH.RemoveVideoAPI)
		})

		// User management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			// JSON API routes for users (admin only)
			r.Post("/api/users", userH.CreateUserAPI)
		})
	})

	// Video streaming — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm)).Get("/v/{id}", streamH.ServeVideo)

	// HLS streaming — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm)).Get("/v/{id}/hls/*", streamH.ServeHLS)

	// Download — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm)).Get("/v/{id}/download", streamH.ServeDownload)

	// Resource detail — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm)).Get("/api/resources/{id}", resourceH.GetResourceAPI)

	// SPA catch-all — serve the single-page application for all unmatched routes.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			respondJSONError(w, "Not found", http.StatusNotFound)
			return
		}
		spa, err := web.SPA()
		if err != nil {
			slog.Error("failed to read SPA", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(spa)
	})

	return r, stops
}
