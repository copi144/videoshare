package model

import (
	"context"
	"database/sql"
	"time"

	"videoshare/database"
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
	q  *database.Queries
}

// NewPlaylistStore creates a new PlaylistStore.
func NewPlaylistStore(db *sql.DB) *PlaylistStore {
	return &PlaylistStore{db: db, q: database.New(db)}
}

// Insert creates a new playlist.
func (s *PlaylistStore) Insert(p *Playlist) error {
	ctx := context.Background()
	return s.q.CreatePlaylist(ctx, database.CreatePlaylistParams{
		ID:          p.ID,
		CategoryID:  p.CategoryID,
		Name:        p.Name,
		Description: p.Description,
		CreatedBy:   p.CreatedBy,
		SortOrder:   int64(p.SortOrder),
	})
}

// GetByID retrieves a playlist by ID.
func (s *PlaylistStore) GetByID(id string) (*Playlist, error) {
	ctx := context.Background()
	p, err := s.q.GetPlaylist(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Playlist{
		ID:          p.ID,
		CategoryID:  p.CategoryID,
		Name:        p.Name,
		Description: p.Description,
		CreatedBy:   p.CreatedBy,
		SortOrder:   int(p.SortOrder),
		CreatedAt:   p.CreatedAt,
	}, nil
}

// ListByCategory returns playlists in a category.
func (s *PlaylistStore) ListByCategory(categoryID string) ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListPlaylistsByCategory(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			ID:          p.ID,
			CategoryID:  p.CategoryID,
			Name:        p.Name,
			Description: p.Description,
			CreatedBy:   p.CreatedBy,
			SortOrder:   int(p.SortOrder),
			CreatedAt:   p.CreatedAt,
		})
	}
	return playlists, nil
}

// AddVideo adds a video to a playlist.
func (s *PlaylistStore) AddVideo(playlistID, resourceID string, sortOrder int) error {
	ctx := context.Background()
	return s.q.AddVideoToPlaylist(ctx, database.AddVideoToPlaylistParams{
		PlaylistID: playlistID,
		ResourceID: resourceID,
		SortOrder:  int64(sortOrder),
	})
}

// RemoveVideo removes a video from a playlist.
func (s *PlaylistStore) RemoveVideo(playlistID, resourceID string) error {
	ctx := context.Background()
	return s.q.RemoveVideoFromPlaylist(ctx, database.RemoveVideoFromPlaylistParams{
		PlaylistID: playlistID,
		ResourceID: resourceID,
	})
}

// ListVideos returns resource IDs in a playlist, ordered by sort_order.
func (s *PlaylistStore) ListVideos(playlistID string) ([]string, error) {
	ctx := context.Background()
	return s.q.ListPlaylistVideos(ctx, playlistID)
}

// Delete removes a playlist.
func (s *PlaylistStore) Delete(id string) error {
	ctx := context.Background()
	return s.q.DeletePlaylist(ctx, id)
}

// ListAll returns all playlists ordered by sort_order then creation date.
func (s *PlaylistStore) ListAll() ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListAllPlaylists(ctx)
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			ID:          p.ID,
			CategoryID:  p.CategoryID,
			Name:        p.Name,
			Description: p.Description,
			CreatedBy:   p.CreatedBy,
			SortOrder:   int(p.SortOrder),
			CreatedAt:   p.CreatedAt,
		})
	}
	return playlists, nil
}

// ListPaginated returns a page of playlists ordered by sort_order then creation date.
func (s *PlaylistStore) ListPaginated(limit, offset int) ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListPlaylistsPaginated(ctx, database.ListPlaylistsPaginatedParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			ID:          p.ID,
			CategoryID:  p.CategoryID,
			Name:        p.Name,
			Description: p.Description,
			CreatedBy:   p.CreatedBy,
			SortOrder:   int(p.SortOrder),
			CreatedAt:   p.CreatedAt,
		})
	}
	return playlists, nil
}

// Count returns the total number of playlists.
func (s *PlaylistStore) Count() (int, error) {
	ctx := context.Background()
	count, err := s.q.CountPlaylists(ctx)
	return int(count), err
}

// GetPlaylistsForResource returns playlist IDs that a resource belongs to.
func (s *PlaylistStore) GetPlaylistsForResource(resourceID string) ([]string, error) {
	ctx := context.Background()
	return s.q.GetPlaylistsForResource(ctx, resourceID)
}
