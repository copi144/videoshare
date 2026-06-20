package model

import (
	"context"
	"database/sql"
	"regexp"
	"time"

	"videoshare/database"
)

var validName = regexp.MustCompile(`^[0-9A-Za-z\-]+$`)

// IsValidName checks that a name matches the allowed pattern [0-9A-Za-z-]+.
func IsValidName(name string) bool {
	return validName.MatchString(name)
}

// IsGlobal reports whether the given category name is the special Global category.
func IsGlobal(categoryName string) bool {
	return IsGlobalCategoryName(categoryName)
}

// IsPublic reports whether videos in this category are publicly accessible without a password.
func IsPublic(categoryName string) bool {
	return IsGlobal(categoryName)
}

// Category represents a video category.
type Category struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// CategoryStore provides CRUD operations for categories.
type CategoryStore struct {
	db *sql.DB
	q  *database.Queries
}

// NewCategoryStore creates a new CategoryStore.
func NewCategoryStore(db *sql.DB) *CategoryStore {
	return &CategoryStore{db: db, q: database.New(db)}
}

// Insert creates a new category.
func (s *CategoryStore) Insert(c *Category) error {
	ctx := context.Background()
	return s.q.CreateCategory(ctx, database.CreateCategoryParams{
		Name:        c.Name,
		DisplayName: c.DisplayName,
		Description: c.Description,
		CreatedBy:   c.CreatedBy,
	})
}

// GetByName retrieves a category by name.
func (s *CategoryStore) GetByName(name string) (*Category, error) {
	ctx := context.Background()
	c, err := s.q.GetCategory(ctx, name)
	if err != nil {
		return nil, err
	}
	return &Category{
		Name:        c.Name,
		DisplayName: c.DisplayName,
		Description: c.Description,
		CreatedBy:   c.CreatedBy,
		CreatedAt:   c.CreatedAt,
	}, nil
}

// List returns all categories.
func (s *CategoryStore) List() ([]*Category, error) {
	ctx := context.Background()
	items, err := s.q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	cats := make([]*Category, 0, len(items))
	for _, c := range items {
		cats = append(cats, &Category{
			Name:        c.Name,
			DisplayName: c.DisplayName,
			Description: c.Description,
			CreatedBy:   c.CreatedBy,
			CreatedAt:   c.CreatedAt,
		})
	}
	return cats, nil
}

// ListByUploader returns categories that a specific uploader is assigned to.
func (s *CategoryStore) ListByUploader(name string) ([]*Category, error) {
	ctx := context.Background()
	items, err := s.q.ListCategoriesByUploader(ctx, name)
	if err != nil {
		return nil, err
	}
	cats := make([]*Category, 0, len(items))
	for _, c := range items {
		cats = append(cats, &Category{
			Name:        c.Name,
			DisplayName: c.DisplayName,
			Description: c.Description,
			CreatedBy:   c.CreatedBy,
			CreatedAt:   c.CreatedAt,
		})
	}
	return cats, nil
}

// ListPaginated returns a page of categories ordered by creation date descending.
func (s *CategoryStore) ListPaginated(limit, offset int) ([]*Category, error) {
	ctx := context.Background()
	items, err := s.q.ListCategoriesPaginated(ctx, database.ListCategoriesPaginatedParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}
	cats := make([]*Category, 0, len(items))
	for _, c := range items {
		cats = append(cats, &Category{
			Name:        c.Name,
			DisplayName: c.DisplayName,
			Description: c.Description,
			CreatedBy:   c.CreatedBy,
			CreatedAt:   c.CreatedAt,
		})
	}
	return cats, nil
}

// Count returns the total number of categories.
func (s *CategoryStore) Count() (int, error) {
	ctx := context.Background()
	count, err := s.q.CountCategories(ctx)
	return int(count), err
}

// ListByUploaderPaginated returns a page of categories that a specific uploader is assigned to.
func (s *CategoryStore) ListByUploaderPaginated(name string, limit, offset int) ([]*Category, error) {
	ctx := context.Background()
	items, err := s.q.ListCategoriesByUploaderPaginated(ctx, database.ListCategoriesByUploaderPaginatedParams{
		Name:   name,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}
	cats := make([]*Category, 0, len(items))
	for _, c := range items {
		cats = append(cats, &Category{
			Name:        c.Name,
			DisplayName: c.DisplayName,
			Description: c.Description,
			CreatedBy:   c.CreatedBy,
			CreatedAt:   c.CreatedAt,
		})
	}
	return cats, nil
}

// CountByUploader returns the total number of categories that a specific uploader is assigned to.
func (s *CategoryStore) CountByUploader(name string) (int, error) {
	ctx := context.Background()
	count, err := s.q.CountCategoriesByUploader(ctx, name)
	return int(count), err
}

// Member represents a user assigned to a category with upload permission status.
type Member struct {
	Name      string
	CanUpload bool
}

// AssignUploaders sets the members for a category (replaces all existing).
func (s *CategoryStore) AssignUploaders(categoryName string, members []Member) error {
	ctx := context.Background()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)
	if err := qtx.ClearCategoryUploaders(ctx, categoryName); err != nil {
		return err
	}
	for _, m := range members {
		canUpload := int64(0)
		if m.CanUpload {
			canUpload = 1
		}
		if err := qtx.AddUploader(ctx, database.AddUploaderParams{
			CategoryName: categoryName,
			Name:         m.Name,
			CanUpload:    canUpload,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetUploaders returns uploaders assigned to a category.
func (s *CategoryStore) GetUploaders(categoryName string) ([]database.ListUploadersRow, error) {
	ctx := context.Background()
	return s.q.ListUploaders(ctx, categoryName)
}

// CanUpload checks if a user is authorized to upload to a category.
func (s *CategoryStore) CanUpload(userName, categoryName string) (bool, error) {
	ctx := context.Background()
	count, err := s.q.CanUpload(ctx, database.CanUploadParams{
		CategoryName: categoryName,
		Name:         userName,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsAssigned checks if a user is assigned to a category (regardless of can_upload).
func (s *CategoryStore) IsAssigned(userName, categoryName string) (bool, error) {
	ctx := context.Background()
	count, err := s.q.IsAssigned(ctx, database.IsAssignedParams{
		CategoryName: categoryName,
		Name:         userName,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetVideoCount returns the number of resources in a given category.
func (s *CategoryStore) GetVideoCount(categoryName string) (int, error) {
	ctx := context.Background()
	count, err := s.q.GetCategoryVideoCount(ctx, categoryName)
	return int(count), err
}

// Delete removes a category.
func (s *CategoryStore) Delete(name string) error {
	ctx := context.Background()
	return s.q.DeleteCategory(ctx, name)
}
