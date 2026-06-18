package model

import (
	"database/sql"
	"time"
)

// User represents a system user.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "admin" or "uploader"
	CreatedAt    time.Time `json:"created_at"`
}

// UserStore provides CRUD operations for users.
type UserStore struct {
	db *sql.DB
}

// NewUserStore creates a new UserStore.
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

// Insert creates a new user record.
func (s *UserStore) Insert(u *User) error {
	query := `INSERT INTO users (id, username, password_hash, role) VALUES (?, ?, ?, ?)`
	_, err := s.db.Exec(query, u.ID, u.Username, u.PasswordHash, u.Role)
	return err
}

// GetByID retrieves a user by ID.
func (s *UserStore) GetByID(id string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// GetByUsername retrieves a user by username (for login).
func (s *UserStore) GetByUsername(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?", username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// List returns all users.
func (s *UserStore) List() ([]*User, error) {
	rows, err := s.db.Query("SELECT id, username, password_hash, role, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
