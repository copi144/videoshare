package model

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"time"
)

// ShareLink represents a share link for a category or playlist.
type ShareLink struct {
	ID             string     `json:"id"`
	Password       string     `json:"-"`
	TargetType     string     `json:"target_type"`
	TargetID       string     `json:"target_id"`
	TargetCategory string     `json:"target_category,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedBy      string     `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ShareLinkStore provides CRUD operations for share_links (categories/playlists) with periodic cleanup.
type ShareLinkStore struct {
	db          *sql.DB
	stopCleanup chan struct{}
}

// NewShareLinkStore creates a new ShareLinkStore.
func NewShareLinkStore(db *sql.DB) *ShareLinkStore {
	return &ShareLinkStore{
		db:          db,
		stopCleanup: make(chan struct{}),
	}
}

// StartCleanup launches a background goroutine that deletes expired share links every hour.
func (s *ShareLinkStore) StartCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := s.db.Exec("DELETE FROM share_links WHERE expires_at IS NOT NULL AND expires_at <= ?", time.Now().UTC()); err != nil {
					slog.Error("share_links cleanup error", "error", err)
				}
			case <-s.stopCleanup:
				return
			}
		}
	}()
}

// StopCleanup signals the cleanup goroutine to stop.
func (s *ShareLinkStore) StopCleanup() {
	close(s.stopCleanup)
}

// GenerateShareLinkID generates 8 random bytes → 16 hex chars for use as id or password.
func GenerateShareLinkID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create inserts a new share link.
func (s *ShareLinkStore) Create(link *ShareLink) error {
	_, err := s.db.Exec(
		`INSERT INTO share_links (id, password, target_type, target_id, target_category, expires_at, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		link.ID, link.Password, link.TargetType, link.TargetID, link.TargetCategory, link.ExpiresAt, link.CreatedBy, link.CreatedAt,
	)
	return err
}

// GetByID finds a valid (not expired) share link by ID.
func (s *ShareLinkStore) GetByID(id string) (*ShareLink, error) {
	row := s.db.QueryRow(
		`SELECT id, password, target_type, target_id, target_category, expires_at, created_by, created_at FROM share_links WHERE id = ? AND (expires_at IS NULL OR expires_at > ?)`,
		id, time.Now().UTC(),
	)
	link := &ShareLink{}
	var expiresAt sql.NullTime
	err := row.Scan(&link.ID, &link.Password, &link.TargetType, &link.TargetID, &link.TargetCategory, &expiresAt, &link.CreatedBy, &link.CreatedAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		link.ExpiresAt = &expiresAt.Time
	}
	return link, nil
}

// ListByTarget returns all non-expired share links for a target (category/playlist).
func (s *ShareLinkStore) ListByTarget(targetType, targetID string) ([]*ShareLink, error) {
	rows, err := s.db.Query(
		`SELECT id, password, target_type, target_id, target_category, expires_at, created_by, created_at FROM share_links WHERE target_type = ? AND target_id = ? AND (expires_at IS NULL OR expires_at > ?) ORDER BY created_at DESC`,
		targetType, targetID, time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*ShareLink
	for rows.Next() {
		link := &ShareLink{}
		var expiresAt sql.NullTime
		if err := rows.Scan(&link.ID, &link.Password, &link.TargetType, &link.TargetID, &link.TargetCategory, &expiresAt, &link.CreatedBy, &link.CreatedAt); err != nil {
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
