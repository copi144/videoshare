package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/mdp/qrterminal/v3"

	"videoshare/handler"
	"videoshare/model"
	"videoshare/transcode"
)

func main() {
	cfg := Load()

	// Initialize structured JSON logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	db, err := model.OpenDB(cfg.DataDir)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	// db.Close() called explicitly in ordered shutdown (srv first)

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
	globalAdminID, err := model.GetAdminUserID(db)
	if err != nil {
		slog.Error("failed to lookup admin user ID for global category bootstrap", "error", err)
	} else {
		if err := model.BootstrapGlobalCategory(db, globalAdminID); err != nil {
			slog.Error("failed to bootstrap global category", "error", err)
		}
	}

	sessStore := model.NewSessionStore(db)
	// sessStore.StopCleanup() called explicitly in ordered shutdown (srv first)

	sm := scs.New()
	sm.Store = sessStore
	sm.Lifetime = 30 * time.Minute // sliding expiry: each API call extends session by 30min
	sm.Cookie.Secure = cfg.CookieSecure
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Path = "/v"

	resourceStore := model.NewResourceStore(db)
	userStore := model.NewUserStore(db)
	categoryStore := model.NewCategoryStore(db)
	playlistStore := model.NewPlaylistStore(db)

	// Initialize transcode queue with access to the resource store for status updates.
	tc := transcode.LoadTranscodeConfig()
	tq := transcode.NewQueue(tc, resourceStore)
	// tq.Shutdown() called explicitly in ordered shutdown (srv first)

	// Reset stalled 'processing' jobs to 'pending' on startup.
	transcode.StartupRecovery(resourceStore)

	router, rateLimitStops := handler.NewRouter(sm, resourceStore, cfg.DataDir, db, userStore, categoryStore, playlistStore, tq, cfg.FFmpegPath)

	addr := cfg.Addr
	srv := &http.Server{Addr: addr, Handler: router}

	slog.Info("starting server", "addr", addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	listenErrCh := make(chan error, 1)
	go func() {
		listenErrCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
		stop()

		// ordered shutdown (srv first per spec)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Warn("server shutdown error", "error", err)
		}
		cancel()

		tq.Shutdown()
		sessStore.StopCleanup()
		for _, stopFn := range rateLimitStops {
			stopFn()
		}
		db.Close()

		slog.Info("shutdown complete")
	case err := <-listenErrCh:
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
		// run ordered cleanup on early listen error (e.g. bind) so we exit cleanly
		tq.Shutdown()
		sessStore.StopCleanup()
		for _, stopFn := range rateLimitStops {
			stopFn()
		}
		db.Close()
		if err != nil && err != http.ErrServerClosed {
			os.Exit(1)
		}
	}
}
