package model

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"time"
)

// ShareResource represents a share link for a single resource.
type ShareResource struct {
	ResourceID string     `json:"resource_id"`
	Password   string     `json:"-"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedBy  string     `json:"created_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ShareResourceStore provides CRUD operations for share_resources with periodic cleanup.
type ShareResourceStore struct {
	db          *sql.DB
	stopCleanup chan struct{}
}

// NewShareResourceStore creates a new ShareResourceStore.
func NewShareResourceStore(db *sql.DB) *ShareResourceStore {
	return &ShareResourceStore{
		db:          db,
		stopCleanup: make(chan struct{}),
	}
}

// StartCleanup launches a background goroutine that deletes expired share resources every hour.
func (s *ShareResourceStore) StartCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := s.db.Exec("DELETE FROM share_resources WHERE expires_at IS NOT NULL AND expires_at <= ?", time.Now().UTC()); err != nil {
					slog.Error("share_resources cleanup error", "error", err)
				}
			case <-s.stopCleanup:
				return
			}
		}
	}()
}

// StopCleanup signals the cleanup goroutine to stop.
func (s *ShareResourceStore) StopCleanup() {
	close(s.stopCleanup)
}

// GenerateSharePassword generates 8 random bytes → 16 hex chars.
func GenerateSharePassword() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create inserts a new share resource link.
func (s *ShareResourceStore) Create(link *ShareResource) error {
	_, err := s.db.Exec(
		`INSERT INTO share_resources (resource_id, password, expires_at, created_by, created_at) VALUES (?, ?, ?, ?, ?)`,
		link.ResourceID, link.Password, link.ExpiresAt, link.CreatedBy, link.CreatedAt,
	)
	return err
}

// GetByResourceAndPassword finds a valid (not expired) share resource for the given resource and password.
func (s *ShareResourceStore) GetByResourceAndPassword(resourceID, password string) (*ShareResource, error) {
	row := s.db.QueryRow(
		`SELECT resource_id, password, expires_at, created_by, created_at FROM share_resources WHERE resource_id = ? AND password = ? AND (expires_at IS NULL OR expires_at > ?)`,
		resourceID, password, time.Now().UTC(),
	)
	link := &ShareResource{}
	var expiresAt sql.NullTime
	err := row.Scan(&link.ResourceID, &link.Password, &expiresAt, &link.CreatedBy, &link.CreatedAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		link.ExpiresAt = &expiresAt.Time
	}
	return link, nil
}

// ListByResource returns all non-expired share resources for a resource.
func (s *ShareResourceStore) ListByResource(resourceID string) ([]*ShareResource, error) {
	rows, err := s.db.Query(
		`SELECT resource_id, password, expires_at, created_by, created_at FROM share_resources WHERE resource_id = ? AND (expires_at IS NULL OR expires_at > ?) ORDER BY created_at DESC`,
		resourceID, time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*ShareResource
	for rows.Next() {
		link := &ShareResource{}
		var expiresAt sql.NullTime
		if err := rows.Scan(&link.ResourceID, &link.Password, &expiresAt, &link.CreatedBy, &link.CreatedAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			link.ExpiresAt = &expiresAt.Time
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// Delete removes a share resource link by resource ID and password.
func (s *ShareResourceStore) Delete(resourceID, password string) error {
	_, err := s.db.Exec(`DELETE FROM share_resources WHERE resource_id = ? AND password = ?`, resourceID, password)
	return err
}
