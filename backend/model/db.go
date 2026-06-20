package model

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pquerna/otp/totp"

	_ "modernc.org/sqlite"
)

// GlobalCategoryName is the fixed name for the implicit "Global" category.
// Videos in this category require no password — they are publicly accessible.
const GlobalCategoryName = "global"

// IsGlobalCategoryName reports whether the given name matches the fixed GlobalCategoryName ("global").
func IsGlobalCategoryName(name string) bool {
	return name == GlobalCategoryName
}

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

// BootstrapAdmin ensures an admin user exists. If no user with is_admin=1 is found,
// it creates one with a TOTP key and returns the otpauth:// URI (for QR display).
// Returns an empty string if the admin already exists.
func BootstrapAdmin(db *sql.DB, name string) (totpURI string, err error) {
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&count)
	if err != nil {
		return "", fmt.Errorf("check admin existence: %w", err)
	}

	if count > 0 {
		slog.Debug("admin user already exists, skipping bootstrap")
		return "", nil
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VideoShare",
		AccountName: name,
	})
	if err != nil {
		return "", fmt.Errorf("generate totp key: %w", err)
	}

	_, err = db.Exec(
		"INSERT INTO users (name, totp_secret, display_name, is_admin) VALUES (?, ?, '', 1)",
		name, key.Secret(),
	)
	if err != nil {
		return "", fmt.Errorf("insert admin user: %w", err)
	}

	slog.Info("admin user bootstrapped", "name", name)
	return key.URL(), nil
}

// BootstrapGlobalCategory ensures the Global category exists.
// It uses an idempotent INSERT OR IGNORE so it is safe to call every startup.
func BootstrapGlobalCategory(db *sql.DB, adminName string) error {
	result, err := db.Exec(
		`INSERT OR IGNORE INTO categories (name, display_name, description, created_by)
		 VALUES (?, 'Global', 'Public videos (no password required)', ?)`,
		GlobalCategoryName, adminName,
	)
	if err != nil {
		return fmt.Errorf("bootstrap global category: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		slog.Info("global category bootstrapped", "name", GlobalCategoryName)
	}
	return nil
}

func migrate(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS resources (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		filename TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		content_type TEXT NOT NULL,
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
	columns := []string{"uploaded_by", "transcode_status", "banned", "no_transcode", "resource_type"}
	for _, col := range columns {
		var def string
		switch col {
		case "banned":
			def = "INTEGER NOT NULL DEFAULT 0"
		case "no_transcode":
			def = "INTEGER NOT NULL DEFAULT 0"
		case "transcode_status":
			def = "TEXT NOT NULL DEFAULT 'none'"
		case "resource_type":
			def = "TEXT NOT NULL DEFAULT 'video'"
		default:
			def = "TEXT REFERENCES users(name)"
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
			name TEXT PRIMARY KEY,
			totp_secret TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			is_admin INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			name TEXT PRIMARY KEY,
			display_name TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL REFERENCES users(name),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS category_users (
			category_name TEXT NOT NULL REFERENCES categories(name) ON DELETE CASCADE,
			name TEXT NOT NULL REFERENCES users(name) ON DELETE CASCADE,
			can_upload INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (category_name, name)
		)`,
		`CREATE TABLE IF NOT EXISTS playlists (
			name TEXT NOT NULL,
			category_name TEXT NOT NULL REFERENCES categories(name) ON DELETE CASCADE,
			playlist_type TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			created_by TEXT NOT NULL REFERENCES users(name),
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (category_name, name)
		)`,
		`CREATE TABLE IF NOT EXISTS playlist_videos (
			playlist_category_name TEXT NOT NULL,
			playlist_name TEXT NOT NULL,
			resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
			sort_order INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (playlist_category_name, playlist_name, resource_id),
			FOREIGN KEY (playlist_category_name, playlist_name) REFERENCES playlists(category_name, name) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS api_tokens (
			token TEXT PRIMARY KEY,
			username TEXT NOT NULL REFERENCES users(name),
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

	}

	for _, q := range tables {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	// Drop old share_links table (schema changed)
	if _, err := db.Exec("DROP TABLE IF EXISTS share_links"); err != nil {
		return fmt.Errorf("drop old share_links: %w", err)
	}

	// New share_resources table
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS share_resources (
		resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
		password TEXT NOT NULL,
		expires_at DATETIME,
		created_by TEXT NOT NULL REFERENCES users(name) ON DELETE CASCADE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (resource_id, password)
	)`); err != nil {
		return fmt.Errorf("create share_resources: %w", err)
	}

	// New share_links table (categories/playlists)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS share_links (
		id TEXT PRIMARY KEY,
		password TEXT NOT NULL,
		target_type TEXT NOT NULL,
		target_id TEXT NOT NULL,
		expires_at DATETIME,
		created_by TEXT NOT NULL REFERENCES users(name) ON DELETE CASCADE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create share_links: %w", err)
	}

	// Resource-categories join table (many-to-many)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS resource_categories (
		resource_id TEXT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
		category_name TEXT NOT NULL REFERENCES categories(name) ON DELETE CASCADE,
		PRIMARY KEY (resource_id, category_name)
	)`); err != nil {
		return fmt.Errorf("create resource_categories: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_resource_categories_resource_id ON resource_categories(resource_id)"); err != nil {
		return fmt.Errorf("create resource_categories resource_id index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_resource_categories_category_name ON resource_categories(category_name)"); err != nil {
		return fmt.Errorf("create resource_categories category_name index: %w", err)
	}

	// Explicit performance indexes (added for pagination and list queries; idempotent)
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_resources_created_at ON resources(created_at)"); err != nil {
		return fmt.Errorf("create resources created_at index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_resources_uploaded_by ON resources(uploaded_by)"); err != nil {
		return fmt.Errorf("create resources uploaded_by index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_resources_transcode_status ON resources(transcode_status)"); err != nil {
		return fmt.Errorf("create resources transcode_status index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_resources_banned ON resources(banned)"); err != nil {
		return fmt.Errorf("create resources banned index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_resources_no_transcode ON resources(no_transcode)"); err != nil {
		return fmt.Errorf("create resources no_transcode index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_categories_created_at ON categories(created_at)"); err != nil {
		return fmt.Errorf("create categories created_at index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_playlists_category_name ON playlists(category_name)"); err != nil {
		return fmt.Errorf("create playlists category_name index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_playlists_created_at ON playlists(created_at)"); err != nil {
		return fmt.Errorf("create playlists created_at index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_playlists_sort_order ON playlists(sort_order)"); err != nil {
		return fmt.Errorf("create playlists sort_order index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_resources_resource_type ON resources(resource_type)"); err != nil {
		return fmt.Errorf("create resources resource_type index: %w", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_playlists_playlist_type ON playlists(playlist_type)"); err != nil {
		return fmt.Errorf("create playlists playlist_type index: %w", err)
	}

	// Indexes for playlist_videos resource_id lookups (GetPlaylistsForResource)
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_playlist_videos_resource_id ON playlist_videos(resource_id)"); err != nil {
		return fmt.Errorf("create playlist_videos resource_id index: %w", err)
	}

	// Index for category_users name lookups (ListCategoriesByUploader)
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_category_users_name ON category_users(name)"); err != nil {
		return fmt.Errorf("create category_users name index: %w", err)
	}

	// Composite index for share_links target lookups (ListByTarget)
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_share_links_target ON share_links(target_type, target_id)"); err != nil {
		return fmt.Errorf("create share_links target index: %w", err)
	}

	return nil
}


