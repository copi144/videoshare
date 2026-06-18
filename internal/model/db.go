package model

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

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

func migrate(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS resources (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
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

	adminQuery := `CREATE TABLE IF NOT EXISTS admin_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(adminQuery); err != nil {
		return fmt.Errorf("create admin_config table: %w", err)
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

	return nil
}
