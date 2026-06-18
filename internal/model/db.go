package model

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	_ "modernc.org/sqlite"
)

// GlobalCategoryID is the fixed ID for the implicit "Global" category.
// Videos in this category require no password — they are publicly accessible.
const GlobalCategoryID = "global"

// OpenDB opens a SQLite database at the given path, applies WAL mode,
// sets connection limits, and runs auto-migration.
func OpenDB(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "videoshare.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// BootstrapAdmin ensures an admin user exists. If no user with role='admin' is found,
// it creates one with a TOTP key and returns the otpauth:// URI (for QR display).
// Returns an empty string if the admin already exists.
func BootstrapAdmin(db *sql.DB, username string) (totpURI string, err error) {
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return "", fmt.Errorf("check admin existence: %w", err)
	}

	if count > 0 {
		slog.Debug("admin user already exists, skipping bootstrap")
		return "", nil
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VideoShare",
		AccountName: username,
	})
	if err != nil {
		return "", fmt.Errorf("generate totp key: %w", err)
	}

	id := uuid.New().String()
	_, err = db.Exec(
		"INSERT INTO users (id, username, totp_secret, role) VALUES (?, ?, ?, 'admin')",
		id, username, key.Secret(),
	)
	if err != nil {
		return "", fmt.Errorf("insert admin user: %w", err)
	}

	slog.Info("admin user bootstrapped", "username", username)
	return key.URL(), nil
}

// BootstrapGlobalCategory ensures the Global category exists.
// It uses an idempotent INSERT OR IGNORE so it is safe to call every startup.
func BootstrapGlobalCategory(db *sql.DB, adminUserID string) error {
	result, err := db.Exec(
		`INSERT OR IGNORE INTO categories (id, name, description, created_by)
		 VALUES (?, 'Global', 'Public videos (no password required)', ?)`,
		GlobalCategoryID, adminUserID,
	)
	if err != nil {
		return fmt.Errorf("bootstrap global category: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		slog.Info("global category bootstrapped", "id", GlobalCategoryID)
	}
	return nil
}

func migrate(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS resources (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		password_hash TEXT NOT NULL,
		filename TEXT NOT NULL DEFAULT '',
		file_size INTEGER NOT NULL DEFAULT 0,
		content_type TEXT NOT NULL DEFAULT 'video/mp4',
		views INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("create resources table: %w", err)
	}

	sessionQuery := `CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		data BLOB NOT NULL,
		expiry DATETIME NOT NULL
	)`

	if _, err := db.Exec(sessionQuery); err != nil {
		return fmt.Errorf("create sessions table: %w", err)
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expiry)"); err != nil {
		return fmt.Errorf("create sessions expiry index: %w", err)
	}

	// Add new columns to resources table (idempotent — errors are expected if columns exist)
	columns := []string{"uploaded_by", "category_id", "transcode_status"}
	for _, col := range columns {
		def := "TEXT"
		if col == "transcode_status" {
			def = "TEXT NOT NULL DEFAULT 'none'"
		} else {
			def = "TEXT REFERENCES users(id)"
		}
		_, err := db.Exec(fmt.Sprintf("ALTER TABLE resources ADD COLUMN %s %s", col, def))
		if err != nil {
			// Column already exists or other error — log and continue
			slog.Debug("column may already exist", "column", col, "error", err)
		}
	}

	// New multi-user tables
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			totp_secret TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'uploader',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL REFERENCES users(id),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS category_uploaders (
			category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			PRIMARY KEY (category_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS playlists (
			id TEXT PRIMARY KEY,
			category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL REFERENCES users(id),
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS playlist_videos (
			playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
			resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
			sort_order INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (playlist_id, resource_id)
		)`,
	}

	for _, q := range tables {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	return nil
}


