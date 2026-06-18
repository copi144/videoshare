package model

import (
	"database/sql"
	"fmt"
	"time"
)

// AdminStore manages admin configuration.
type AdminStore struct {
	db *sql.DB
}

// NewAdminStore creates a new AdminStore.
func NewAdminStore(db *sql.DB) *AdminStore {
	return &AdminStore{db: db}
}

// SetPassword stores the bcrypt-hashed upload password.
func (s *AdminStore) SetPassword(hash string) error {
	_, err := s.db.Exec(
		`INSERT INTO admin_config (key, value, updated_at) VALUES ('upload_password', ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?`,
		hash, time.Now(), hash, time.Now(),
	)
	return err
}

// GetPassword retrieves the stored bcrypt password hash. Returns empty string if not set.
func (s *AdminStore) GetPassword() (string, error) {
	row := s.db.QueryRow("SELECT value FROM admin_config WHERE key = 'upload_password'")
	var hash string
	if err := row.Scan(&hash); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("get password: %w", err)
	}
	return hash, nil
}

// PasswordIsSet returns true if an upload password has been configured.
func (s *AdminStore) PasswordIsSet() (bool, error) {
	hash, err := s.GetPassword()
	return hash != "", err
}
