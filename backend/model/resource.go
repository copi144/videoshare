package model

import (
	"database/sql"
	"fmt"
	"time"
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
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ResourceStore provides CRUD operations for resources.
type ResourceStore struct {
	db *sql.DB
}

// NewResourceStore creates a new ResourceStore.
func NewResourceStore(db *sql.DB) *ResourceStore {
	return &ResourceStore{db: db}
}

// Insert creates a new resource record.
func (s *ResourceStore) Insert(r *Resource) error {
	query := `INSERT INTO resources (id, title, password_hash, filename, file_size, content_type, uploaded_by, category_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		r.ID, r.Title, r.PasswordHash,
		r.Filename, r.FileSize, r.ContentType,
		r.UploadedBy, r.CategoryID,
	)
	return err
}

// GetByID retrieves a resource by its ID.
func (s *ResourceStore) GetByID(id string) (*Resource, error) {
	query := `SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, created_at, updated_at
		FROM resources WHERE id = ?`

	r := &Resource{}
	err := s.db.QueryRow(query, id).Scan(
		&r.ID, &r.Title, &r.PasswordHash,
		&r.Filename, &r.FileSize, &r.ContentType,
		&r.Views, &r.UploadedBy, &r.CategoryID,
		&r.TranscodeStatus, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// List returns all resources ordered by creation date descending.
func (s *ResourceStore) List() ([]*Resource, error) {
	query := `SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, created_at, updated_at
		FROM resources ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*Resource
	for rows.Next() {
		r := &Resource{}
		if err := rows.Scan(
			&r.ID, &r.Title, &r.PasswordHash,
			&r.Filename, &r.FileSize, &r.ContentType,
			&r.Views, &r.UploadedBy, &r.CategoryID,
			&r.TranscodeStatus, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// ListByUploader returns all resources uploaded by a specific user, ordered by creation date descending.
func (s *ResourceStore) ListByUploader(userID string) ([]*Resource, error) {
	query := `SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, created_at, updated_at
		FROM resources WHERE uploaded_by = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*Resource
	for rows.Next() {
		r := &Resource{}
		if err := rows.Scan(
			&r.ID, &r.Title, &r.PasswordHash,
			&r.Filename, &r.FileSize, &r.ContentType,
			&r.Views, &r.UploadedBy, &r.CategoryID,
			&r.TranscodeStatus, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// ListPaginated returns a page of resources ordered by creation date descending.
func (s *ResourceStore) ListPaginated(limit, offset int) ([]*Resource, error) {
	query := `SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, created_at, updated_at
		FROM resources ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*Resource
	for rows.Next() {
		r := &Resource{}
		if err := rows.Scan(
			&r.ID, &r.Title, &r.PasswordHash,
			&r.Filename, &r.FileSize, &r.ContentType,
			&r.Views, &r.UploadedBy, &r.CategoryID,
			&r.TranscodeStatus, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// Count returns the total number of resources.
func (s *ResourceStore) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM resources").Scan(&count)
	return count, err
}

// ListByUploaderPaginated returns a page of resources uploaded by a specific user, ordered by creation date descending.
func (s *ResourceStore) ListByUploaderPaginated(userID string, limit, offset int) ([]*Resource, error) {
	query := `SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, created_at, updated_at
		FROM resources WHERE uploaded_by = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*Resource
	for rows.Next() {
		r := &Resource{}
		if err := rows.Scan(
			&r.ID, &r.Title, &r.PasswordHash,
			&r.Filename, &r.FileSize, &r.ContentType,
			&r.Views, &r.UploadedBy, &r.CategoryID,
			&r.TranscodeStatus, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// CountByUploader returns the total number of resources uploaded by a specific user.
func (s *ResourceStore) CountByUploader(userID string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM resources WHERE uploaded_by = ?", userID).Scan(&count)
	return count, err
}

// Delete removes a resource by its ID.
func (s *ResourceStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM resources WHERE id = ?", id)
	return err
}

// DeleteWithFile deletes a resource record and prepares for file cleanup in a transaction.
// The fileCleanup callback is called within the transaction for atomicity.
// Duplicate detection uses content BLAKE3 hash as the resource ID (PK). The row and files are removed on delete, freeing the hash for re-upload of identical content.
func (s *ResourceStore) DeleteWithFile(id string, fileCleanup func() error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM resources WHERE id = ?", id)
	if err != nil {
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
	_, err := s.db.Exec("UPDATE resources SET transcode_status = ? WHERE id = ?", status, id)
	return err
}

// ListByTranscodeStatus returns all resources with the given transcode status.
func (s *ResourceStore) ListByTranscodeStatus(status string) ([]*Resource, error) {
	query := `SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, created_at, updated_at
		FROM resources WHERE transcode_status = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*Resource
	for rows.Next() {
		r := &Resource{}
		if err := rows.Scan(
			&r.ID, &r.Title, &r.PasswordHash,
			&r.Filename, &r.FileSize, &r.ContentType,
			&r.Views, &r.UploadedBy, &r.CategoryID,
			&r.TranscodeStatus, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// IncrementViews increases the view count for a resource.
func (s *ResourceStore) IncrementViews(id string) error {
	_, err := s.db.Exec("UPDATE resources SET views = views + 1 WHERE id = ?", id)
	return err
}
