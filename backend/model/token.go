package model

import (
	"database/sql"
	"time"
)

// APIToken represents a stored API token for cookie-free API authentication.
type APIToken struct {
	Token     string
	UserID    string
	UserRole  string
	Username  string
	CreatedAt time.Time
}

// SaveAPIToken inserts a new API token into the database.
func SaveAPIToken(db *sql.DB, token, userID, role, username string) error {
	_, err := db.Exec(
		"INSERT INTO api_tokens (token, user_id, user_role, username) VALUES (?, ?, ?, ?)",
		token, userID, role, username,
	)
	return err
}

// GetAPIToken retrieves an API token record by its token value.
// Returns nil and an error if the token is not found.
func GetAPIToken(db *sql.DB, token string) (*APIToken, error) {
	t := &APIToken{}
	err := db.QueryRow(
		"SELECT token, user_id, user_role, username, created_at FROM api_tokens WHERE token = ?",
		token,
	).Scan(&t.Token, &t.UserID, &t.UserRole, &t.Username, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteAPIToken removes an API token from the database.
func DeleteAPIToken(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM api_tokens WHERE token = ?", token)
	return err
}
