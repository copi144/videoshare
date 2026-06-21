package handler

import (
	"database/sql"
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
	shareResourceStore *model.ShareResourceStore, shareLinkStore *model.ShareLinkStore,
	transcodeQueue *transcode.Queue, ffmpegPath string) (http.Handler, []func()) {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(sm.LoadAndSave)

	r.Use(middleware.APIAuth(db, sm))

	rl, stop := middleware.RateLimit(300, time.Minute, func(r *http.Request) bool {
		return middleware.IsAPIAuthenticated(r.Context())
	})
	r.Use(rl)

	var stops []func()
	stops = append(stops, stop)

	// Health check
	r.Get("/health", NewHealthHandler(db).ServeHealth)

	// Favicon
	r.Get("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write(web.Favicon())
	})

	// Homepage — serve the SPA
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(web.SPA())
	})

	// Create handlers with dependency injection.
	userH := NewUserHandler(userStore, sm, db)
	resourceH := NewResourceHandler(resourceStore, categoryStore, dataDir, userStore, playlistStore, transcodeQueue, ffmpegPath)
	shareResourceH := NewShareResourceHandler(shareResourceStore, resourceStore)
	shareLinkH := NewShareLinkHandler(shareLinkStore, categoryStore, playlistStore, resourceStore, sm)
	shareScopeMW := middleware.RequireShareScope(sm, shareLinkStore)
	sessionH := NewSessionHandler(userStore, resourceStore, shareResourceStore, categoryStore, sm, db, shareLinkStore)
	streamH := NewStreamHandler(resourceStore, dataDir)
	playlistH := NewPlaylistHandler(playlistStore, resourceStore, categoryStore, sm)
	categoryH := NewCategoryHandler(categoryStore, userStore, sm)

	// JSON API routes that are public
	r.Post("/api/login", userH.ServeLoginAPI)

	// Session management — create or refresh auth session
	r.Post("/api/session", sessionH.ServeSessionAPI)

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
			r.Get("/api/categories/{id}/uploaders", categoryH.ListUploadersAPI)
			r.Post("/api/categories/{id}/uploaders", categoryH.AssignUploadersAPI)
		})

		// Playlist listing — any authenticated user can see them (for browse filters)
		r.Get("/api/playlists", playlistH.ListPlaylistsAPI)

		// Playlist management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			// JSON API routes for playlists (admin only)
			r.Post("/api/playlists", playlistH.CreatePlaylistAPI)
			r.Delete("/api/playlists/{name}", playlistH.DeletePlaylistAPI)
			r.Post("/api/playlists/{name}/videos", playlistH.AddVideoAPI)
			r.Delete("/api/playlists/{name}/videos/{resourceId}", playlistH.RemoveVideoAPI)
		})

		// User management — admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(sm))

			// JSON API routes for users (admin only)
			r.Post("/api/users", userH.CreateUserAPI)
			r.Get("/api/users", userH.ListUsersAPI)
			r.Delete("/api/users/{name}", userH.DeleteUserAPI)
			r.Post("/api/users/{name}/reset-totp", userH.ResetTOTPAPI)
		})

		// Resource share link management — requires user auth
		r.Post("/api/share-resources", shareResourceH.CreateAPI)
		r.Get("/api/share-resources", shareResourceH.ListAPI)
		r.Delete("/api/share-resources/{resourceID}/{password}", shareResourceH.DeleteAPI)

		// Category/Playlist share link management — requires user auth
		r.Post("/api/share-links", shareLinkH.CreateAPI)
		r.Get("/api/share-links", shareLinkH.ListAPI)
		r.Delete("/api/share-links/{id}", shareLinkH.DeleteAPI)
	})

	// Share link auth — public (for /#/s/{id}/{password} URL access)
	r.Post("/api/share-links/{id}/auth", shareLinkH.AuthenticateAPI)
	r.Get("/api/share-links/{id}/resources", shareLinkH.GetSharedResourcesAPI)

	// Video streaming — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm), shareScopeMW).Get("/v/{id}", streamH.ServeVideo)

	// HLS streaming — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm), shareScopeMW).Get("/v/{id}/hls/*", streamH.ServeHLS)

	// Download — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm), shareScopeMW).Get("/v/{id}/download", streamH.ServeDownload)

	// Audio/Image streaming — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm), shareScopeMW).Get("/a/{id}", streamH.ServeAudio)
	r.With(middleware.RequireUserOrVideoAuth(sm), shareScopeMW).Get("/i/{id}", streamH.ServeImage)

	// Resource detail — accessible by both system users and share-link viewers
	r.With(middleware.RequireUserOrVideoAuth(sm), shareScopeMW).Get("/api/resources/{id}", resourceH.GetResourceAPI)

	// SPA catch-all — serve the single-page application for all unmatched routes.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			respondJSONError(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(web.SPA())
	})

	return r, stops
}
