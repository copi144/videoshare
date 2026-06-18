package model

import (
	"database/sql"
	"time"
)

// Playlist represents a video playlist within a category.
type Playlist struct {
	ID          string    `json:"id"`
	CategoryID  string    `json:"category_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// PlaylistStore provides CRUD operations for playlists.
type PlaylistStore struct {
	db *sql.DB
}

// NewPlaylistStore creates a new PlaylistStore.
func NewPlaylistStore(db *sql.DB) *PlaylistStore {
	return &PlaylistStore{db: db}
}

// Insert creates a new playlist.
func (s *PlaylistStore) Insert(p *Playlist) error {
	_, err := s.db.Exec(
		"INSERT INTO playlists (id, category_id, name, description, created_by, sort_order) VALUES (?, ?, ?, ?, ?, ?)",
		p.ID, p.CategoryID, p.Name, p.Description, p.CreatedBy, p.SortOrder,
	)
	return err
}

// GetByID retrieves a playlist by ID.
func (s *PlaylistStore) GetByID(id string) (*Playlist, error) {
	p := &Playlist{}
	err := s.db.QueryRow(
		"SELECT id, category_id, name, description, created_by, sort_order, created_at FROM playlists WHERE id = ?", id,
	).Scan(&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.CreatedBy, &p.SortOrder, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListByCategory returns playlists in a category.
func (s *PlaylistStore) ListByCategory(categoryID string) ([]*Playlist, error) {
	rows, err := s.db.Query(
		"SELECT id, category_id, name, description, created_by, sort_order, created_at FROM playlists WHERE category_id = ? ORDER BY sort_order ASC, created_at ASC",
		categoryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []*Playlist
	for rows.Next() {
		p := &Playlist{}
		if err := rows.Scan(&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.CreatedBy, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}

// AddVideo adds a video to a playlist.
func (s *PlaylistStore) AddVideo(playlistID, resourceID string, sortOrder int) error {
	_, err := s.db.Exec(
		"INSERT INTO playlist_videos (playlist_id, resource_id, sort_order) VALUES (?, ?, ?) ON CONFLICT(playlist_id, resource_id) DO UPDATE SET sort_order = ?",
		playlistID, resourceID, sortOrder, sortOrder,
	)
	return err
}

// RemoveVideo removes a video from a playlist.
func (s *PlaylistStore) RemoveVideo(playlistID, resourceID string) error {
	_, err := s.db.Exec("DELETE FROM playlist_videos WHERE playlist_id = ? AND resource_id = ?", playlistID, resourceID)
	return err
}

// ListVideos returns resource IDs in a playlist, ordered by sort_order.
func (s *PlaylistStore) ListVideos(playlistID string) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT resource_id FROM playlist_videos WHERE playlist_id = ? ORDER BY sort_order ASC",
		playlistID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Delete removes a playlist.
func (s *PlaylistStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM playlists WHERE id = ?", id)
	return err
}

// ListAll returns all playlists ordered by sort_order then creation date.
func (s *PlaylistStore) ListAll() ([]*Playlist, error) {
	rows, err := s.db.Query(
		"SELECT id, category_id, name, description, created_by, sort_order, created_at FROM playlists ORDER BY sort_order ASC, created_at ASC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []*Playlist
	for rows.Next() {
		p := &Playlist{}
		if err := rows.Scan(&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.CreatedBy, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}

// GetPlaylistsForResource returns playlist IDs that a resource belongs to.
func (s *PlaylistStore) GetPlaylistsForResource(resourceID string) ([]string, error) {
	rows, err := s.db.Query("SELECT playlist_id FROM playlist_videos WHERE resource_id = ?", resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
