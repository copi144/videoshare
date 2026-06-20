package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/mdp/qrterminal/v3"
	"github.com/pquerna/otp/totp"

	"videoshare/handler"
	"videoshare/model"
	"videoshare/transcode"
)

func main() {
	cfg := Load()

	// Initialize structured JSON logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
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
		fmt.Printf("  Name: %s\n", cfg.AdminUsername)
		fmt.Println("  Scan the QR code below with your")
		fmt.Println("  authenticator app (Google Authenticator, Authy, etc.)")
		fmt.Println("  Or enter the URI manually in your browser")
		fmt.Printf("  TOTP URI: %s\n", totpURI)
		fmt.Println("═══════════════════════════════════════════")
		qrterminal.Generate(totpURI, qrterminal.L, os.Stdout)
		fmt.Println("")
	}

	// Bootstrap the global category (public/no-password videos).
	adminName, err := model.GetAdminName(db)
	if err != nil {
		slog.Error("failed to lookup admin name for global category bootstrap", "error", err)
	} else {
		if err := model.BootstrapGlobalCategory(db, adminName); err != nil {
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
	sm.Cookie.Path = "/"

	resourceStore := model.NewResourceStore(db)
	userStore := model.NewUserStore(db)
	categoryStore := model.NewCategoryStore(db)
	playlistStore := model.NewPlaylistStore(db)

	// Check for TOTP reset file
	resetPath := filepath.Join(cfg.DataDir, "reset-admin-totp.txt")
	if data, err := os.ReadFile(resetPath); err == nil {
		adminName := strings.TrimSpace(string(data))
		if adminName == "" {
			adminName = cfg.AdminUsername
		}

		// Find the admin user by name
		adminUser, err := userStore.GetByName(adminName)
		if err != nil {
			slog.Error("failed to find admin user for TOTP reset", "name", adminName, "error", err)
		} else {
			key, err := totp.Generate(totp.GenerateOpts{
				Issuer:      "VideoShare",
				AccountName: adminUser.Name,
			})
			if err != nil {
				slog.Error("failed to generate TOTP key for reset", "error", err)
			} else {
				// Update TOTP secret directly
				_, err := db.Exec("UPDATE users SET totp_secret = ? WHERE name = ?", key.Secret(), adminUser.Name)
				if err != nil {
					slog.Error("failed to reset TOTP secret", "error", err)
				} else {
					fmt.Println("\n═══════════════════════════════════════════")
					fmt.Println("  Admin TOTP Reset!")
					fmt.Printf("  Name: %s\n", adminName)
					fmt.Println("  Scan the QR code below with your")
					fmt.Println("  authenticator app")
					fmt.Printf("  TOTP URI: %s\n", key.URL())
					fmt.Println("═══════════════════════════════════════════")
					qrterminal.Generate(key.URL(), qrterminal.L, os.Stdout)
					fmt.Println("")

					// Remove the reset file
					if err := os.Remove(resetPath); err != nil {
						slog.Warn("failed to remove TOTP reset file", "path", resetPath, "error", err)
					}
				}
			}
		}
	}

	// Initialize transcode queue with access to the resource store for status updates.
	tc := transcode.LoadTranscodeConfig()
	tq := transcode.NewQueue(tc, resourceStore, cfg.DataDir)
	// tq.Shutdown() called explicitly in ordered shutdown (srv first)

	// Reset stalled 'processing' jobs to 'pending' on startup.
	transcode.StartupRecovery(resourceStore)

	shareResourceStore := model.NewShareResourceStore(db)
	shareLinkStore := model.NewShareLinkStore(db)
	router, rateLimitStops := handler.NewRouter(sm, resourceStore, cfg.DataDir, db, userStore, categoryStore, playlistStore, shareResourceStore, shareLinkStore, tq, cfg.FFmpegPath)

	// Start cleanup goroutines
	shareResourceStore.StartCleanup()
	shareLinkStore.StartCleanup()
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := model.DeleteExpiredAPITokens(db); err != nil {
					slog.Error("api token cleanup error", "error", err)
				}
			}
		}
	}()

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
		shareResourceStore.StopCleanup()
		shareLinkStore.StopCleanup()
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
		shareResourceStore.StopCleanup()
		shareLinkStore.StopCleanup()
		for _, stopFn := range rateLimitStops {
			stopFn()
		}
		db.Close()
		if err != nil && err != http.ErrServerClosed {
			os.Exit(1)
		}
	}
}
