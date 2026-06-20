package model

import (
	"database/sql"
	"time"
)

// APIToken represents a stored API token for cookie-free API authentication.
type APIToken struct {
	Token     string
	Name      string
	IsAdmin   bool
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SaveAPIToken inserts a new API token into the database.
func SaveAPIToken(db *sql.DB, token, name string, expiresAt time.Time) error {
	_, err := db.Exec(
		"INSERT INTO api_tokens (token, username, expires_at) VALUES (?, ?, ?)",
		token, name, expiresAt,
	)
	return err
}

// GetAPIToken retrieves a non-expired API token record by its token value.
// Returns nil and an error if the token is not found or has expired.
func GetAPIToken(db *sql.DB, token string) (*APIToken, error) {
	t := &APIToken{}
	err := db.QueryRow(
		"SELECT a.token, a.username, u.is_admin, a.expires_at, a.created_at FROM api_tokens a JOIN users u ON a.username = u.name WHERE a.token = ? AND a.expires_at > ?",
		token, time.Now().UTC(),
	).Scan(&t.Token, &t.Name, &t.IsAdmin, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

const APITokenTTL = 30 * time.Minute

// RefreshAPITokenExpiry updates the expires_at for a valid token, sliding it forward.
func RefreshAPITokenExpiry(db *sql.DB, token string) error {
	_, err := db.Exec("UPDATE api_tokens SET expires_at = ? WHERE token = ?", time.Now().UTC().Add(APITokenTTL), token)
	return err
}

// DeleteAPIToken removes an API token from the database.
func DeleteAPIToken(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM api_tokens WHERE token = ?", token)
	return err
}

// DeleteExpiredAPITokens removes all API tokens whose expiry has passed.
func DeleteExpiredAPITokens(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM api_tokens WHERE expires_at <= ?", time.Now().UTC())
	return err
}
