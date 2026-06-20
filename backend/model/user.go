package model

import (
	"context"
	"database/sql"
	"time"

	"videoshare/database"
)

// User represents a system user.
type User struct {
	Name        string    `json:"name"`
	TotpSecret  string    `json:"-"`
	DisplayName string    `json:"display_name"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserStore provides CRUD operations for users.
type UserStore struct {
	db *sql.DB
	q  *database.Queries
}

// NewUserStore creates a new UserStore.
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db, q: database.New(db)}
}

// Insert creates a new user record.
func (s *UserStore) Insert(u *User) error {
	ctx := context.Background()
	isAdmin := int64(0)
	if u.IsAdmin {
		isAdmin = 1
	}
	return s.q.CreateUser(ctx, database.CreateUserParams{
		Name:        u.Name,
		TotpSecret:  u.TotpSecret,
		DisplayName: u.DisplayName,
		IsAdmin:     isAdmin,
	})
}

// GetByName retrieves a user by name.
func (s *UserStore) GetByName(name string) (*User, error) {
	ctx := context.Background()
	u, err := s.q.GetUser(ctx, name)
	if err != nil {
		return nil, err
	}
	return &User{
		Name:        u.Name,
		TotpSecret:  u.TotpSecret,
		DisplayName: u.DisplayName,
		IsAdmin:     u.IsAdmin == 1,
		CreatedAt:   u.CreatedAt,
	}, nil
}

// List returns all users.
func (s *UserStore) List() ([]*User, error) {
	ctx := context.Background()
	items, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]*User, 0, len(items))
	for _, u := range items {
		users = append(users, &User{
			Name:        u.Name,
			TotpSecret:  u.TotpSecret,
			DisplayName: u.DisplayName,
			IsAdmin:     u.IsAdmin == 1,
			CreatedAt:   u.CreatedAt,
		})
	}
	return users, nil
}

// GetAdminName returns the name of one admin user (used for global category bootstrap).
func GetAdminName(db *sql.DB) (string, error) {
	ctx := context.Background()
	q := database.New(db)
	return q.GetAdminName(ctx)
}

// Delete removes a user by name.
func (s *UserStore) Delete(name string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE name = ?", name)
	return err
}

// UpdateTotpSecret updates the TOTP secret for a user.
func (s *UserStore) UpdateTotpSecret(name, secret string) error {
	_, err := s.db.Exec("UPDATE users SET totp_secret = ? WHERE name = ?", secret, name)
	return err
}
