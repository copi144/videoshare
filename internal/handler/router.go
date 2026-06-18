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
	resourceStore *model.ResourceStore, dataDir string, db *sql.DB) http.Handler {
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
	authH := NewAuthHandler(resourceStore, sm, templates)

	// General login page
	r.Get("/login", authH.ServeLoginPage)

	// Public routes — password entry for shared videos (tighter rate limit)
	r.Route("/s/{id}", func(r chi.Router) {
		r.Use(middleware.RateLimit(5, time.Minute))

		r.Get("/", authH.ServeSharePage)
		r.Post("/auth", authH.Authenticate)

		// Watch page requires session authentication
		r.Group(func(r chi.Router) {
			r.Use(middleware.SessionAuth(sm))
			r.Get("/watch", authH.ServeWatchPage)
		})
	})

	// Admin area — requires session authentication
	resourceH := NewResourceHandler(resourceStore, templates, dataDir)
	streamH := NewStreamHandler(resourceStore, dataDir)

	r.Group(func(r chi.Router) {
		r.Use(middleware.SessionAuth(sm))

		// Admin pages
		r.Get("/admin", resourceH.List)

		// Upload endpoint
		r.Post("/api/upload", resourceH.Upload)

		// Resource management
		r.Delete("/api/resource/{id}", resourceH.Delete)

		// Video streaming (authenticated)
		r.Get("/api/video/{id}", streamH.ServeVideo)
	})

	// Static file serving for embedded assets
	// TODO: serve embedded static files from web directory

	return r
}
