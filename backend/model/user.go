package model

import (
	"context"
	"database/sql"
	"time"

	"videoshare/database"
)

// User represents a system user.
type User struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	TotpSecret string    `json:"-"`
	Role       string    `json:"role"` // "admin" or "uploader"
	CreatedAt  time.Time `json:"created_at"`
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
	return s.q.CreateUser(ctx, database.CreateUserParams{
		ID:         u.ID,
		Username:   u.Username,
		TotpSecret: u.TotpSecret,
		Role:       u.Role,
	})
}

// GetByID retrieves a user by ID.
func (s *UserStore) GetByID(id string) (*User, error) {
	ctx := context.Background()
	u, err := s.q.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:         u.ID,
		Username:   u.Username,
		TotpSecret: u.TotpSecret,
		Role:       u.Role,
		CreatedAt:  u.CreatedAt,
	}, nil
}

// GetByUsername retrieves a user by username (for login).
func (s *UserStore) GetByUsername(username string) (*User, error) {
	ctx := context.Background()
	u, err := s.q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:         u.ID,
		Username:   u.Username,
		TotpSecret: u.TotpSecret,
		Role:       u.Role,
		CreatedAt:  u.CreatedAt,
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
			ID:         u.ID,
			Username:   u.Username,
			TotpSecret: u.TotpSecret,
			Role:       u.Role,
			CreatedAt:  u.CreatedAt,
		})
	}
	return users, nil
}

// GetAdminUserID returns the ID of one admin user (used for global category bootstrap).
func GetAdminUserID(db *sql.DB) (string, error) {
	var id string
	err := db.QueryRow("SELECT id FROM users WHERE role = 'admin' LIMIT 1").Scan(&id)
	return id, err
}
