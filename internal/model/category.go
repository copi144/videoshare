package model

import (
	"database/sql"
	"regexp"
	"time"
)

var validCategoryName = regexp.MustCompile(`^[0-9A-Za-z\-]+$`)

func IsValidCategoryName(name string) bool {
	return validCategoryName.MatchString(name)
}

// Category represents a video category.
type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// CategoryStore provides CRUD operations for categories.
type CategoryStore struct {
	db *sql.DB
}

// NewCategoryStore creates a new CategoryStore.
func NewCategoryStore(db *sql.DB) *CategoryStore {
	return &CategoryStore{db: db}
}

// Insert creates a new category.
func (s *CategoryStore) Insert(c *Category) error {
	_, err := s.db.Exec(
		"INSERT INTO categories (id, name, description, created_by) VALUES (?, ?, ?, ?)",
		c.ID, c.Name, c.Description, c.CreatedBy,
	)
	return err
}

// GetByID retrieves a category by ID.
func (s *CategoryStore) GetByID(id string) (*Category, error) {
	c := &Category{}
	err := s.db.QueryRow(
		"SELECT id, name, description, created_by, created_at FROM categories WHERE id = ?", id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedBy, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// List returns all categories.
func (s *CategoryStore) List() ([]*Category, error) {
	rows, err := s.db.Query("SELECT id, name, description, created_by, created_at FROM categories ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []*Category
	for rows.Next() {
		c := &Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// ListByUploader returns categories that a specific uploader is assigned to.
func (s *CategoryStore) ListByUploader(userID string) ([]*Category, error) {
	query := `SELECT c.id, c.name, c.description, c.created_by, c.created_at
		FROM categories c
		JOIN category_uploaders cu ON cu.category_id = c.id
		WHERE cu.user_id = ?
		ORDER BY c.created_at DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []*Category
	for rows.Next() {
		c := &Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// AssignUploaders sets the uploaders for a category (replaces all existing).
func (s *CategoryStore) AssignUploaders(categoryID string, userIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM category_uploaders WHERE category_id = ?", categoryID)
	if err != nil {
		return err
	}

	for _, uid := range userIDs {
		_, err = tx.Exec("INSERT INTO category_uploaders (category_id, user_id) VALUES (?, ?)", categoryID, uid)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetUploaders returns user IDs assigned to a category.
func (s *CategoryStore) GetUploaders(categoryID string) ([]string, error) {
	rows, err := s.db.Query("SELECT user_id FROM category_uploaders WHERE category_id = ?", categoryID)
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

// IsUploaderAuthorized checks if a user is assigned to upload to a category.
func (s *CategoryStore) IsUploaderAuthorized(userID, categoryID string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM category_uploaders WHERE category_id = ? AND user_id = ?",
		categoryID, userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetVideoCount returns the number of resources in a given category.
func (s *CategoryStore) GetVideoCount(categoryID string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM resources WHERE category_id = ?", categoryID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Delete removes a category.
func (s *CategoryStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM categories WHERE id = ?", id)
	return err
}
