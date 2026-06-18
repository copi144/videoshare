package main

import (
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"

	"videoshare/internal/config"
	"videoshare/internal/handler"
	"videoshare/internal/model"
	"videoshare/internal/web"
)

func main() {
	cfg := config.Load()

	// Initialize structured JSON logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	db, err := model.OpenDB(cfg.DataDir)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	sessStore := model.NewSessionStore(db)
	defer sessStore.StopCleanup()

	sm := scs.New()
	sm.Store = sessStore
	sm.Lifetime = 24 * time.Hour // session expires after 24 hours
	sm.Cookie.Secure = cfg.CookieSecure
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode

	// Decode hex-encoded CSRF key from config.
	csrfKey, err := hex.DecodeString(cfg.CsrfKey)
	if err != nil {
		slog.Error("invalid CSRF_KEY (must be hex-encoded)", "error", err)
		os.Exit(1)
	}

	resourceStore := model.NewResourceStore(db)

	router := handler.NewRouter(sm, web.Templates(), csrfKey, cfg.CookieSecure, resourceStore, cfg.DataDir, db)

	addr := cfg.Addr
	slog.Info("starting server", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
