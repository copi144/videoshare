package model

import (
	"database/sql"
	"fmt"
	"time"
)

// Resource represents a video resource stored in the database.
type Resource struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	PasswordHash string    `json:"-"`
	Filename     string    `json:"filename"`
	FileSize     int64     `json:"file_size"`
	ContentType  string    `json:"content_type"`
	Views        int       `json:"views"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
	query := `INSERT INTO resources (id, title, description, password_hash, filename, file_size, content_type)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		r.ID, r.Title, r.Description, r.PasswordHash,
		r.Filename, r.FileSize, r.ContentType,
	)
	return err
}

// GetByID retrieves a resource by its ID.
func (s *ResourceStore) GetByID(id string) (*Resource, error) {
	query := `SELECT id, title, description, password_hash, filename, file_size, content_type, views, created_at, updated_at
		FROM resources WHERE id = ?`

	r := &Resource{}
	err := s.db.QueryRow(query, id).Scan(
		&r.ID, &r.Title, &r.Description, &r.PasswordHash,
		&r.Filename, &r.FileSize, &r.ContentType,
		&r.Views, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// List returns all resources ordered by creation date descending.
func (s *ResourceStore) List() ([]*Resource, error) {
	query := `SELECT id, title, description, password_hash, filename, file_size, content_type, views, created_at, updated_at
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
			&r.ID, &r.Title, &r.Description, &r.PasswordHash,
			&r.Filename, &r.FileSize, &r.ContentType,
			&r.Views, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// Delete removes a resource by its ID.
func (s *ResourceStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM resources WHERE id = ?", id)
	return err
}

// DeleteWithFile deletes a resource record and prepares for file cleanup in a transaction.
// The fileCleanup callback is called within the transaction for atomicity.
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

// IncrementViews increases the view count for a resource.
func (s *ResourceStore) IncrementViews(id string) error {
	_, err := s.db.Exec("UPDATE resources SET views = views + 1 WHERE id = ?", id)
	return err
}
