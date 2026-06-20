package model

import (
	"context"
	"database/sql"
	"time"

	"videoshare/database"
)

// Playlist represents a playlist within a category.
type Playlist struct {
	Name         string    `json:"name"`
	CategoryName string    `json:"category_name"`
	PlaylistType string    `json:"playlist_type"`
	DisplayName  string    `json:"display_name"`
	Description  string    `json:"description"`
	CreatedBy    string    `json:"created_by"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
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
		CategoryName: p.CategoryName,
		PlaylistType: p.PlaylistType,
		Name:         p.Name,
		DisplayName:  p.DisplayName,
		Description:  p.Description,
		CreatedBy:    p.CreatedBy,
		SortOrder:    int64(p.SortOrder),
	})
}

// GetByName retrieves a playlist by category name and playlist name.
func (s *PlaylistStore) GetByName(categoryName, name string) (*Playlist, error) {
	ctx := context.Background()
	p, err := s.q.GetPlaylist(ctx, database.GetPlaylistParams{
		CategoryName: categoryName,
		Name:         name,
	})
	if err != nil {
		return nil, err
	}
	return &Playlist{
		Name:         p.Name,
		CategoryName: p.CategoryName,
		PlaylistType: p.PlaylistType,
		DisplayName:  p.DisplayName,
		Description:  p.Description,
		CreatedBy:    p.CreatedBy,
		SortOrder:    int(p.SortOrder),
		CreatedAt:    p.CreatedAt,
	}, nil
}

// ListByCategory returns playlists in a category.
func (s *PlaylistStore) ListByCategory(categoryName string) ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListPlaylistsByCategory(ctx, categoryName)
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			Name:         p.Name,
			CategoryName: p.CategoryName,
			PlaylistType: p.PlaylistType,
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			CreatedBy:    p.CreatedBy,
			SortOrder:    int(p.SortOrder),
			CreatedAt:    p.CreatedAt,
		})
	}
	return playlists, nil
}

// AddVideo adds a video to a playlist.
func (s *PlaylistStore) AddVideo(playlistCategoryName, playlistName, resourceID string, sortOrder int) error {
	ctx := context.Background()
	return s.q.AddVideoToPlaylist(ctx, database.AddVideoToPlaylistParams{
		PlaylistCategoryName: playlistCategoryName,
		PlaylistName:         playlistName,
		ResourceID:           resourceID,
		SortOrder:            int64(sortOrder),
	})
}

// RemoveVideo removes a video from a playlist.
func (s *PlaylistStore) RemoveVideo(playlistCategoryName, playlistName, resourceID string) error {
	ctx := context.Background()
	return s.q.RemoveVideoFromPlaylist(ctx, database.RemoveVideoFromPlaylistParams{
		PlaylistCategoryName: playlistCategoryName,
		PlaylistName:         playlistName,
		ResourceID:           resourceID,
	})
}

// ListVideos returns resource IDs in a playlist, ordered by sort_order.
func (s *PlaylistStore) ListVideos(playlistCategoryName, playlistName string) ([]string, error) {
	ctx := context.Background()
	rows, err := s.q.ListPlaylistVideos(ctx, database.ListPlaylistVideosParams{
		PlaylistCategoryName: playlistCategoryName,
		PlaylistName:         playlistName,
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// Delete removes a playlist.
func (s *PlaylistStore) Delete(categoryName, name string) error {
	ctx := context.Background()
	return s.q.DeletePlaylist(ctx, database.DeletePlaylistParams{
		CategoryName: categoryName,
		Name:         name,
	})
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
			Name:         p.Name,
			CategoryName: p.CategoryName,
			PlaylistType: p.PlaylistType,
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			CreatedBy:    p.CreatedBy,
			SortOrder:    int(p.SortOrder),
			CreatedAt:    p.CreatedAt,
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
			Name:         p.Name,
			CategoryName: p.CategoryName,
			PlaylistType: p.PlaylistType,
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			CreatedBy:    p.CreatedBy,
			SortOrder:    int(p.SortOrder),
			CreatedAt:    p.CreatedAt,
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

// GetPlaylistsForResource returns (category, name) pairs for playlists containing a resource.
func (s *PlaylistStore) GetPlaylistsForResource(resourceID string) ([]database.GetPlaylistsForResourceRow, error) {
	ctx := context.Background()
	return s.q.GetPlaylistsForResource(ctx, resourceID)
}

// ListByType returns all playlists with the given type, ordered by sort_order then creation date.
func (s *PlaylistStore) ListByType(playlistType string) ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListPlaylistsByType(ctx, playlistType)
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			Name:         p.Name,
			CategoryName: p.CategoryName,
			PlaylistType: p.PlaylistType,
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			CreatedBy:    p.CreatedBy,
			SortOrder:    int(p.SortOrder),
			CreatedAt:    p.CreatedAt,
		})
	}
	return playlists, nil
}

// ListByTypePaginated returns a page of playlists with the given type, ordered by sort_order then creation date.
func (s *PlaylistStore) ListByTypePaginated(playlistType string, limit, offset int) ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListPlaylistsByTypePaginated(ctx, database.ListPlaylistsByTypePaginatedParams{
		PlaylistType: playlistType,
		Limit:        int64(limit),
		Offset:       int64(offset),
	})
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			Name:         p.Name,
			CategoryName: p.CategoryName,
			PlaylistType: p.PlaylistType,
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			CreatedBy:    p.CreatedBy,
			SortOrder:    int(p.SortOrder),
			CreatedAt:    p.CreatedAt,
		})
	}
	return playlists, nil
}

// CountByType returns the total number of playlists with the given type.
func (s *PlaylistStore) CountByType(playlistType string) (int, error) {
	ctx := context.Background()
	count, err := s.q.CountPlaylistsByType(ctx, playlistType)
	return int(count), err
}

// ListByCategoryAndType returns playlists in a category with the given type.
func (s *PlaylistStore) ListByCategoryAndType(categoryName, playlistType string) ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListPlaylistsByCategoryAndType(ctx, database.ListPlaylistsByCategoryAndTypeParams{
		CategoryName: categoryName,
		PlaylistType: playlistType,
	})
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			Name:         p.Name,
			CategoryName: p.CategoryName,
			PlaylistType: p.PlaylistType,
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			CreatedBy:    p.CreatedBy,
			SortOrder:    int(p.SortOrder),
			CreatedAt:    p.CreatedAt,
		})
	}
	return playlists, nil
}

// ListByCategoryPaginated returns a page of playlists in a category.
func (s *PlaylistStore) ListByCategoryPaginated(categoryName string, limit, offset int) ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListPlaylistsByCategoryPaginated(ctx, database.ListPlaylistsByCategoryPaginatedParams{
		CategoryName: categoryName,
		Limit:        int64(limit),
		Offset:       int64(offset),
	})
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			Name:         p.Name,
			CategoryName: p.CategoryName,
			PlaylistType: p.PlaylistType,
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			CreatedBy:    p.CreatedBy,
			SortOrder:    int(p.SortOrder),
			CreatedAt:    p.CreatedAt,
		})
	}
	return playlists, nil
}

// CountByCategory returns the total number of playlists in a category.
func (s *PlaylistStore) CountByCategory(categoryName string) (int, error) {
	ctx := context.Background()
	count, err := s.q.CountPlaylistsByCategory(ctx, categoryName)
	return int(count), err
}

// ListByCategoryAndTypePaginated returns a page of playlists in a category with the given type.
func (s *PlaylistStore) ListByCategoryAndTypePaginated(categoryName, playlistType string, limit, offset int) ([]*Playlist, error) {
	ctx := context.Background()
	items, err := s.q.ListPlaylistsByCategoryAndTypePaginated(ctx, database.ListPlaylistsByCategoryAndTypePaginatedParams{
		CategoryName: categoryName,
		PlaylistType: playlistType,
		Limit:        int64(limit),
		Offset:       int64(offset),
	})
	if err != nil {
		return nil, err
	}
	playlists := make([]*Playlist, 0, len(items))
	for _, p := range items {
		playlists = append(playlists, &Playlist{
			Name:         p.Name,
			CategoryName: p.CategoryName,
			PlaylistType: p.PlaylistType,
			DisplayName:  p.DisplayName,
			Description:  p.Description,
			CreatedBy:    p.CreatedBy,
			SortOrder:    int(p.SortOrder),
			CreatedAt:    p.CreatedAt,
		})
	}
	return playlists, nil
}

// CountByCategoryAndType returns the total number of playlists in a category with the given type.
func (s *PlaylistStore) CountByCategoryAndType(categoryName, playlistType string) (int, error) {
	ctx := context.Background()
	count, err := s.q.CountPlaylistsByCategoryAndType(ctx, database.CountPlaylistsByCategoryAndTypeParams{
		CategoryName: categoryName,
		PlaylistType: playlistType,
	})
	return int(count), err
}
