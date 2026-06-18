package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/mdp/qrterminal/v3"

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

	// Bootstrap admin user on first run — generates TOTP key.
	totpURI, err := model.BootstrapAdmin(db, cfg.AdminUsername)
	if err != nil {
		slog.Error("failed to bootstrap admin", "error", err)
		os.Exit(1)
	}
	if totpURI != "" {
		// First boot — display TOTP setup credentials in terminal.
		fmt.Println("\n═══════════════════════════════════════════")
		fmt.Println("  Admin Account Created!")
		fmt.Printf("  Username: %s\n", cfg.AdminUsername)
		fmt.Println("  Scan the QR code below with your")
		fmt.Println("  authenticator app (Google Authenticator, Authy, etc.)")
		fmt.Println("  Or enter the URI manually in your browser")
		fmt.Printf("  TOTP URI: %s\n", totpURI)
		fmt.Println("═══════════════════════════════════════════")
		qrterminal.Generate(totpURI, qrterminal.L, os.Stdout)
		fmt.Println("")
	}

	// Bootstrap the global category (public/no-password videos).
	var globalAdminID string
	err = db.QueryRow("SELECT id FROM users WHERE role = 'admin' LIMIT 1").Scan(&globalAdminID)
	if err != nil {
		slog.Error("failed to lookup admin user ID for global category bootstrap", "error", err)
	} else {
		if err := model.BootstrapGlobalCategory(db, globalAdminID); err != nil {
			slog.Error("failed to bootstrap global category", "error", err)
		}
	}

	sessStore := model.NewSessionStore(db)
	defer sessStore.StopCleanup()

	sm := scs.New()
	sm.Store = sessStore
	sm.Lifetime = 24 * time.Hour // session expires after 24 hours
	sm.Cookie.Secure = cfg.CookieSecure
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode

	resourceStore := model.NewResourceStore(db)
	userStore := model.NewUserStore(db)
	categoryStore := model.NewCategoryStore(db)
	playlistStore := model.NewPlaylistStore(db)

	router := handler.NewRouter(sm, web.Templates(), resourceStore, cfg.DataDir, db, userStore, categoryStore, playlistStore)

	addr := cfg.Addr
	slog.Info("starting server", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
