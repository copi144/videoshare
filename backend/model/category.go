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

// IsGlobal reports whether the given category ID is the special Global category.
func IsGlobal(categoryID string) bool {
	return IsGlobalCategoryID(categoryID)
}

// IsPublic reports whether videos in this category are publicly accessible without a password.
func IsPublic(categoryID string) bool {
	return IsGlobal(categoryID)
}

// RequiresPassword reports whether a per-video password is required for this category.
func RequiresPassword(categoryID string) bool {
	return !IsGlobal(categoryID)
}

// Category represents a video category.
//
// Category.ID is the user-supplied validated name (slug) and is used directly as the primary key.
// Renaming a category produces a new ID.
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
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		CreatedBy:   c.CreatedBy,
	})
}

// GetByID retrieves a category by ID.
func (s *CategoryStore) GetByID(id string) (*Category, error) {
	ctx := context.Background()
	c, err := s.q.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Category{
		ID:          c.ID,
		Name:        c.Name,
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
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			CreatedBy:   c.CreatedBy,
			CreatedAt:   c.CreatedAt,
		})
	}
	return cats, nil
}

// ListByUploader returns categories that a specific uploader is assigned to.
func (s *CategoryStore) ListByUploader(userID string) ([]*Category, error) {
	ctx := context.Background()
	items, err := s.q.ListCategoriesByUploader(ctx, userID)
	if err != nil {
		return nil, err
	}
	cats := make([]*Category, 0, len(items))
	for _, c := range items {
		cats = append(cats, &Category{
			ID:          c.ID,
			Name:        c.Name,
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
			ID:          c.ID,
			Name:        c.Name,
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
func (s *CategoryStore) ListByUploaderPaginated(userID string, limit, offset int) ([]*Category, error) {
	ctx := context.Background()
	items, err := s.q.ListCategoriesByUploaderPaginated(ctx, database.ListCategoriesByUploaderPaginatedParams{
		UserID: userID,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}
	cats := make([]*Category, 0, len(items))
	for _, c := range items {
		cats = append(cats, &Category{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			CreatedBy:   c.CreatedBy,
			CreatedAt:   c.CreatedAt,
		})
	}
	return cats, nil
}

// CountByUploader returns the total number of categories that a specific uploader is assigned to.
func (s *CategoryStore) CountByUploader(userID string) (int, error) {
	ctx := context.Background()
	count, err := s.q.CountCategoriesByUploader(ctx, userID)
	return int(count), err
}

// AssignUploaders sets the uploaders for a category (replaces all existing).
func (s *CategoryStore) AssignUploaders(categoryID string, userIDs []string) error {
	ctx := context.Background()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)
	if err := qtx.ClearCategoryUploaders(ctx, categoryID); err != nil {
		return err
	}
	for _, uid := range userIDs {
		if err := qtx.AddCategoryUploader(ctx, database.AddCategoryUploaderParams{
			CategoryID: categoryID,
			UserID:     uid,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetUploaders returns user IDs assigned to a category.
func (s *CategoryStore) GetUploaders(categoryID string) ([]string, error) {
	ctx := context.Background()
	return s.q.GetCategoryUploaders(ctx, categoryID)
}

// IsUploaderAuthorized checks if a user is assigned to upload to a category.
func (s *CategoryStore) IsUploaderAuthorized(userID, categoryID string) (bool, error) {
	ctx := context.Background()
	count, err := s.q.IsUploaderAuthorized(ctx, database.IsUploaderAuthorizedParams{
		CategoryID: categoryID,
		UserID:     userID,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetVideoCount returns the number of resources in a given category.
func (s *CategoryStore) GetVideoCount(categoryID string) (int, error) {
	ctx := context.Background()
	count, err := s.q.GetCategoryVideoCount(ctx, sql.NullString{String: categoryID, Valid: categoryID != ""})
	return int(count), err
}

// Delete removes a category.
func (s *CategoryStore) Delete(id string) error {
	ctx := context.Background()
	return s.q.DeleteCategory(ctx, id)
}
