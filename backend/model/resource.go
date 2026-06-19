package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"videoshare/database"
)

// Resource represents a video resource stored in the database.
type Resource struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	PasswordHash     string    `json:"-"`
	Filename         string    `json:"filename"`
	FileSize         int64     `json:"file_size"`
	ContentType      string    `json:"content_type"`
	Views            int       `json:"views"`
	UploadedBy       string    `json:"uploaded_by"`
	CategoryID       string    `json:"category_id"`
	TranscodeStatus  string    `json:"transcode_status"`
	Banned           bool      `json:"banned"`
	NoTranscode      bool      `json:"no_transcode"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ResourceStore provides CRUD operations for resources.
type ResourceStore struct {
	db *sql.DB
	q  *database.Queries
}

// NewResourceStore creates a new ResourceStore.
func NewResourceStore(db *sql.DB) *ResourceStore {
	return &ResourceStore{db: db, q: database.New(db)}
}

// Insert creates a new resource record.
func (s *ResourceStore) Insert(r *Resource) error {
	ctx := context.Background()
	noTranscode := int64(0)
	if r.NoTranscode {
		noTranscode = 1
	}
	return s.q.CreateResource(ctx, database.CreateResourceParams{
		ID:           r.ID,
		Title:        r.Title,
		PasswordHash: r.PasswordHash,
		Filename:     r.Filename,
		FileSize:     r.FileSize,
		ContentType:  r.ContentType,
		UploadedBy:   sql.NullString{String: r.UploadedBy, Valid: r.UploadedBy != ""},
		CategoryID:   sql.NullString{String: r.CategoryID, Valid: r.CategoryID != ""},
		NoTranscode:  noTranscode,
	})
}

// GetByID retrieves a resource by its ID.
func (s *ResourceStore) GetByID(id string) (*Resource, error) {
	ctx := context.Background()
	r, err := s.q.GetResource(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Resource{
		ID:              r.ID,
		Title:           r.Title,
		PasswordHash:    r.PasswordHash,
		Filename:        r.Filename,
		FileSize:        r.FileSize,
		ContentType:     r.ContentType,
		Views:           int(r.Views),
		UploadedBy:      r.UploadedBy.String,
		CategoryID:      r.CategoryID.String,
		TranscodeStatus: r.TranscodeStatus,
		Banned:          r.Banned != 0,
		NoTranscode:     r.NoTranscode != 0,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}, nil
}

// List returns all resources ordered by creation date descending.
func (s *ResourceStore) List() ([]*Resource, error) {
	ctx := context.Background()
	items, err := s.q.ListResources(ctx)
	if err != nil {
		return nil, err
	}
	resources := make([]*Resource, 0, len(items))
	for _, r := range items {
		resources = append(resources, &Resource{
			ID:              r.ID,
			Title:           r.Title,
			PasswordHash:    r.PasswordHash,
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy.String,
			CategoryID:      r.CategoryID.String,
			TranscodeStatus: r.TranscodeStatus,
			Banned:          r.Banned != 0,
			NoTranscode:     r.NoTranscode != 0,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		})
	}
	return resources, nil
}

// ListByUploader returns all resources uploaded by a specific user, ordered by creation date descending.
func (s *ResourceStore) ListByUploader(userID string) ([]*Resource, error) {
	ctx := context.Background()
	items, err := s.q.ListResourcesByUploader(ctx, sql.NullString{String: userID, Valid: userID != ""})
	if err != nil {
		return nil, err
	}
	resources := make([]*Resource, 0, len(items))
	for _, r := range items {
		resources = append(resources, &Resource{
			ID:              r.ID,
			Title:           r.Title,
			PasswordHash:    r.PasswordHash,
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy.String,
			CategoryID:      r.CategoryID.String,
			TranscodeStatus: r.TranscodeStatus,
			Banned:          r.Banned != 0,
			NoTranscode:     r.NoTranscode != 0,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		})
	}
	return resources, nil
}

// ListPaginated returns a page of resources ordered by creation date descending.
func (s *ResourceStore) ListPaginated(limit, offset int) ([]*Resource, error) {
	ctx := context.Background()
	items, err := s.q.ListResourcesPaginated(ctx, database.ListResourcesPaginatedParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}
	resources := make([]*Resource, 0, len(items))
	for _, r := range items {
		resources = append(resources, &Resource{
			ID:              r.ID,
			Title:           r.Title,
			PasswordHash:    r.PasswordHash,
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy.String,
			CategoryID:      r.CategoryID.String,
			TranscodeStatus: r.TranscodeStatus,
			Banned:          r.Banned != 0,
			NoTranscode:     r.NoTranscode != 0,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		})
	}
	return resources, nil
}

// Count returns the total number of resources.
func (s *ResourceStore) Count() (int, error) {
	ctx := context.Background()
	count, err := s.q.CountResources(ctx)
	return int(count), err
}

// ListByUploaderPaginated returns a page of resources uploaded by a specific user, ordered by creation date descending.
func (s *ResourceStore) ListByUploaderPaginated(userID string, limit, offset int) ([]*Resource, error) {
	ctx := context.Background()
	items, err := s.q.ListResourcesByUploaderPaginated(ctx, database.ListResourcesByUploaderPaginatedParams{
		UploadedBy: sql.NullString{String: userID, Valid: userID != ""},
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		return nil, err
	}
	resources := make([]*Resource, 0, len(items))
	for _, r := range items {
		resources = append(resources, &Resource{
			ID:              r.ID,
			Title:           r.Title,
			PasswordHash:    r.PasswordHash,
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy.String,
			CategoryID:      r.CategoryID.String,
			TranscodeStatus: r.TranscodeStatus,
			Banned:          r.Banned != 0,
			NoTranscode:     r.NoTranscode != 0,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		})
	}
	return resources, nil
}

// CountByUploader returns the total number of resources uploaded by a specific user.
func (s *ResourceStore) CountByUploader(userID string) (int, error) {
	ctx := context.Background()
	count, err := s.q.CountResourcesByUploader(ctx, sql.NullString{String: userID, Valid: userID != ""})
	return int(count), err
}

// Delete removes a resource by its ID.
func (s *ResourceStore) Delete(id string) error {
	ctx := context.Background()
	return s.q.DeleteResource(ctx, id)
}

// DeleteWithFile deletes a resource record and prepares for file cleanup in a transaction.
// The fileCleanup callback is called within the transaction for atomicity.
// Duplicate detection uses content BLAKE3 hash as the resource ID (PK). The row and files are removed on delete, freeing the hash for re-upload of identical content.
func (s *ResourceStore) DeleteWithFile(id string, fileCleanup func() error) error {
	ctx := context.Background()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)
	if err := qtx.DeleteResource(ctx, id); err != nil {
		return fmt.Errorf("delete resource: %w", err)
	}

	if fileCleanup != nil {
		if err := fileCleanup(); err != nil {
			return fmt.Errorf("file cleanup: %w", err)
		}
	}

	return tx.Commit()
}

// UpdateTranscodeStatus updates the transcode status for a resource.
func (s *ResourceStore) UpdateTranscodeStatus(id, status string) error {
	ctx := context.Background()
	return s.q.UpdateTranscodeStatus(ctx, database.UpdateTranscodeStatusParams{
		TranscodeStatus: status,
		ID:              id,
	})
}

// SetBanned sets the banned status for a resource.
func (s *ResourceStore) SetBanned(id string, banned bool) error {
	ctx := context.Background()
	bannedInt := int64(0)
	if banned {
		bannedInt = 1
	}
	return s.q.UpdateResourceBanned(ctx, database.UpdateResourceBannedParams{
		Banned: bannedInt,
		ID:     id,
	})
}

// ListByTranscodeStatus returns all resources with the given transcode status.
func (s *ResourceStore) ListByTranscodeStatus(status string) ([]*Resource, error) {
	ctx := context.Background()
	items, err := s.q.ListResourcesByTranscodeStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	resources := make([]*Resource, 0, len(items))
	for _, r := range items {
		resources = append(resources, &Resource{
			ID:              r.ID,
			Title:           r.Title,
			PasswordHash:    r.PasswordHash,
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy.String,
			CategoryID:      r.CategoryID.String,
			TranscodeStatus: r.TranscodeStatus,
			Banned:          r.Banned != 0,
			NoTranscode:     r.NoTranscode != 0,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		})
	}
	return resources, nil
}

// IncrementViews increases the view count for a resource.
func (s *ResourceStore) IncrementViews(id string) error {
	ctx := context.Background()
	return s.q.IncrementResourceViews(ctx, id)
}
