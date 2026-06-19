package model

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

// ShareLink represents an ephemeral share link with a plaintext password.
type ShareLink struct {
	ID         string     `json:"id"`
	ResourceID string     `json:"resource_id"`
	Password   string     `json:"-"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedBy  string     `json:"created_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ShareLinkStore provides CRUD operations for share_links.
type ShareLinkStore struct {
	db *sql.DB
}

// NewShareLinkStore creates a new ShareLinkStore.
func NewShareLinkStore(db *sql.DB) *ShareLinkStore {
	return &ShareLinkStore{db: db}
}

// GenerateShareLinkPassword generates 8 random bytes and returns a 16-char hex string.
func GenerateShareLinkPassword() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateShareLinkID creates a unique ID for a share link (16 hex chars).
func GenerateShareLinkID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create inserts a new share link. The password must already be generated.
func (s *ShareLinkStore) Create(link *ShareLink) error {
	_, err := s.db.Exec(
		`INSERT INTO share_links (id, resource_id, password, expires_at, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		link.ID, link.ResourceID, link.Password, link.ExpiresAt, link.CreatedBy, link.CreatedAt,
	)
	return err
}

// GetByResourceAndPassword finds a valid (not expired) share link for the given resource and password.
func (s *ShareLinkStore) GetByResourceAndPassword(resourceID, password string) (*ShareLink, error) {
	row := s.db.QueryRow(
		`SELECT id, resource_id, password, expires_at, created_by, created_at FROM share_links WHERE resource_id = ? AND password = ? AND (expires_at IS NULL OR expires_at > ?)`,
		resourceID, password, time.Now().UTC(),
	)
	link := &ShareLink{}
	var expiresAt sql.NullTime
	err := row.Scan(&link.ID, &link.ResourceID, &link.Password, &expiresAt, &link.CreatedBy, &link.CreatedAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		link.ExpiresAt = &expiresAt.Time
	}
	return link, nil
}

// ListByResource returns all non-expired share links for a resource.
func (s *ShareLinkStore) ListByResource(resourceID string) ([]*ShareLink, error) {
	rows, err := s.db.Query(
		`SELECT id, resource_id, password, expires_at, created_by, created_at FROM share_links WHERE resource_id = ? AND (expires_at IS NULL OR expires_at > ?) ORDER BY created_at DESC`,
		resourceID, time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*ShareLink
	for rows.Next() {
		link := &ShareLink{}
		var expiresAt sql.NullTime
		if err := rows.Scan(&link.ID, &link.ResourceID, &link.Password, &expiresAt, &link.CreatedBy, &link.CreatedAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			link.ExpiresAt = &expiresAt.Time
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// Delete removes a share link by ID.
func (s *ShareLinkStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM share_links WHERE id = ?`, id)
	return err
}

// DeleteExpired removes all expired share links.
func (s *ShareLinkStore) DeleteExpired() error {
	_, err := s.db.Exec(`DELETE FROM share_links WHERE expires_at IS NOT NULL AND expires_at <= ?`, time.Now().UTC())
	return err
}
