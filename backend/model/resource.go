package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"videoshare/database"
)

// Resource represents a resource stored in the database.
type Resource struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Filename        string    `json:"filename"`
	FileSize        int64     `json:"file_size"`
	ContentType     string    `json:"content_type"`
	ResourceType    string    `json:"resource_type"`
	Views           int       `json:"views"`
	UploadedBy      string    `json:"uploaded_by"`
	Categories      []string  `json:"categories,omitempty"` // populated via join table when requested
	TranscodeStatus string    `json:"transcode_status"`
	Banned          bool      `json:"banned"`
	NoTranscode     bool      `json:"no_transcode"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
		Filename:     r.Filename,
		FileSize:     r.FileSize,
		ContentType:  r.ContentType,
		ResourceType: r.ResourceType,
		UploadedBy:   r.UploadedBy,
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
		Filename:        r.Filename,
		FileSize:        r.FileSize,
		ContentType:     r.ContentType,
		ResourceType:    r.ResourceType,
		Views:           int(r.Views),
		UploadedBy:      r.UploadedBy,
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
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			ResourceType:    r.ResourceType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy,
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
	items, err := s.q.ListResourcesByUploader(ctx, userID)
	if err != nil {
		return nil, err
	}
	resources := make([]*Resource, 0, len(items))
	for _, r := range items {
		resources = append(resources, &Resource{
			ID:              r.ID,
			Title:           r.Title,
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			ResourceType:    r.ResourceType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy,
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
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			ResourceType:    r.ResourceType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy,
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
		UploadedBy: userID,
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
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			ResourceType:    r.ResourceType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy,
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
	count, err := s.q.CountResourcesByUploader(ctx, userID)
	return int(count), err
}

// Delete removes a resource by its ID.
func (s *ResourceStore) Delete(id string) error {
	ctx := context.Background()
	// Clean up resource_categories first.
	if _, err := s.db.Exec("DELETE FROM resource_categories WHERE resource_id = ?", id); err != nil {
		return fmt.Errorf("cleanup resource categories: %w", err)
	}
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

	// Clean up resource_categories within the transaction.
	if _, err := tx.Exec("DELETE FROM resource_categories WHERE resource_id = ?", id); err != nil {
		return fmt.Errorf("cleanup resource categories: %w", err)
	}

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
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			ResourceType:    r.ResourceType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy,
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

// ListByTypePaginated returns a page of resources with the given type, ordered by creation date descending.
func (s *ResourceStore) ListByTypePaginated(resourceType string, limit, offset int) ([]*Resource, error) {
	ctx := context.Background()
	items, err := s.q.ListResourcesByTypePaginated(ctx, database.ListResourcesByTypePaginatedParams{
		ResourceType: resourceType,
		Limit:        int64(limit),
		Offset:       int64(offset),
	})
	if err != nil {
		return nil, err
	}
	resources := make([]*Resource, 0, len(items))
	for _, r := range items {
		resources = append(resources, &Resource{
			ID:              r.ID,
			Title:           r.Title,
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			ResourceType:    r.ResourceType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy,
			TranscodeStatus: r.TranscodeStatus,
			Banned:          r.Banned != 0,
			NoTranscode:     r.NoTranscode != 0,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		})
	}
	return resources, nil
}

// CountByType returns the total number of resources with the given type.
func (s *ResourceStore) CountByType(resourceType string) (int, error) {
	ctx := context.Background()
	count, err := s.q.CountResourcesByType(ctx, resourceType)
	return int(count), err
}

// ListByTypeAndUploaderPaginated returns a page of resources with the given type and uploader.
func (s *ResourceStore) ListByTypeAndUploaderPaginated(resourceType, uploaderID string, limit, offset int) ([]*Resource, error) {
	rows, err := s.db.Query(
		`SELECT id, title, filename, file_size, content_type, resource_type, views, uploaded_by, no_transcode, transcode_status, banned, created_at, updated_at
		 FROM resources WHERE resource_type = ? AND uploaded_by = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		resourceType, uploaderID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*Resource
	for rows.Next() {
		r := &Resource{}
		var filename, contentType, uploadedBy, transcodeStatus string
		var fileSize, views, noTranscode int64
		var banned int64
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&r.ID, &r.Title, &filename, &fileSize, &contentType, &r.ResourceType, &views, &uploadedBy, &noTranscode, &transcodeStatus, &banned, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		r.Filename = filename
		r.FileSize = fileSize
		r.ContentType = contentType
		r.Views = int(views)
		r.UploadedBy = uploadedBy
		r.NoTranscode = noTranscode != 0
		r.TranscodeStatus = transcodeStatus
		r.Banned = banned != 0
		r.CreatedAt = createdAt
		r.UpdatedAt = updatedAt
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// CountByTypeAndUploader returns the total number of resources with the given type and uploader.
func (s *ResourceStore) CountByTypeAndUploader(resourceType, uploaderID string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM resources WHERE resource_type = ? AND uploaded_by = ?", resourceType, uploaderID).Scan(&count)
	return count, err
}

// AddResourceCategory adds a category assignment to a resource (idempotent).
func (s *ResourceStore) AddResourceCategory(resourceID, categoryName string) error {
	ctx := context.Background()
	return s.q.AddResourceCategory(ctx, database.AddResourceCategoryParams{
		ResourceID:   resourceID,
		CategoryName: categoryName,
	})
}

// RemoveResourceFromCategory removes one category assignment from a resource.
func (s *ResourceStore) RemoveResourceFromCategory(resourceID, categoryName string) error {
	ctx := context.Background()
	return s.q.RemoveResourceFromCategory(ctx, database.RemoveResourceFromCategoryParams{
		ResourceID:   resourceID,
		CategoryName: categoryName,
	})
}

// RemoveAllResourceCategories removes all category assignments for a resource.
func (s *ResourceStore) RemoveAllResourceCategories(resourceID string) error {
	ctx := context.Background()
	return s.q.RemoveAllResourceCategories(ctx, resourceID)
}

// GetResourceCategoriesCount returns how many categories a resource belongs to.
func (s *ResourceStore) GetResourceCategoriesCount(resourceID string) (int, error) {
	ctx := context.Background()
	count, err := s.q.GetResourceCategoryCount(ctx, resourceID)
	return int(count), err
}

// GetResourceCategories returns the list of category names a resource belongs to.
func (s *ResourceStore) GetResourceCategories(resourceID string) ([]string, error) {
	ctx := context.Background()
	return s.q.ListResourceCategories(ctx, resourceID)
}

// ListByCategoryPaginated returns a page of resources in a specific category (admin view).
func (s *ResourceStore) ListByCategoryPaginated(categoryName string, limit, offset int) ([]*Resource, error) {
	ctx := context.Background()
	items, err := s.q.ListResourcesByCategoryPaginated(ctx, database.ListResourcesByCategoryPaginatedParams{
		CategoryName: categoryName,
		Limit:        int64(limit),
		Offset:       int64(offset),
	})
	if err != nil {
		return nil, err
	}
	resources := make([]*Resource, 0, len(items))
	for _, r := range items {
		resources = append(resources, &Resource{
			ID:              r.ID,
			Title:           r.Title,
			Filename:        r.Filename,
			FileSize:        r.FileSize,
			ContentType:     r.ContentType,
			ResourceType:    r.ResourceType,
			Views:           int(r.Views),
			UploadedBy:      r.UploadedBy,
			TranscodeStatus: r.TranscodeStatus,
			Banned:          r.Banned != 0,
			NoTranscode:     r.NoTranscode != 0,
			CreatedAt:       r.CreatedAt,
			UpdatedAt:       r.UpdatedAt,
		})
	}
	return resources, nil
}

// ListByCategoryAndUploaderPaginated returns resources in a category uploaded by a specific user.
func (s *ResourceStore) ListByCategoryAndUploaderPaginated(categoryName, uploaderID string, limit, offset int) ([]*Resource, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.title, r.filename, r.file_size, r.content_type, r.resource_type,
		       r.views, r.uploaded_by, r.no_transcode,
		       r.transcode_status, r.banned, r.created_at, r.updated_at
		FROM resources r
		JOIN resource_categories rc ON r.id = rc.resource_id
		WHERE rc.category_name = ? AND r.uploaded_by = ?
		ORDER BY r.created_at DESC LIMIT ? OFFSET ?`,
		categoryName, uploaderID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResources(rows)
}

// CountByCategory returns the total number of resources in a category.
func (s *ResourceStore) CountByCategory(categoryName string) (int, error) {
	ctx := context.Background()
	count, err := s.q.CountResourcesByCategory(ctx, categoryName)
	return int(count), err
}

// CountByCategoryAndUploader returns the count of resources a user has in a category.
func (s *ResourceStore) CountByCategoryAndUploader(categoryName, uploaderID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM resource_categories rc
		 JOIN resources r ON r.id = rc.resource_id
		 WHERE rc.category_name = ? AND r.uploaded_by = ?`,
		categoryName, uploaderID,
	).Scan(&count)
	return count, err
}

// scanResources scans SQL rows into a slice of Resource pointers.
func scanResources(rows *sql.Rows) ([]*Resource, error) {
	var resources []*Resource
	for rows.Next() {
		r := &Resource{}
		var filename, contentType, uploadedBy, transcodeStatus string
		var fileSize, views, noTranscode int64
		var banned int64
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&r.ID, &r.Title, &filename, &fileSize, &contentType, &r.ResourceType, &views, &uploadedBy, &noTranscode, &transcodeStatus, &banned, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		r.Filename = filename
		r.FileSize = fileSize
		r.ContentType = contentType
		r.Views = int(views)
		r.UploadedBy = uploadedBy
		r.NoTranscode = noTranscode != 0
		r.TranscodeStatus = transcodeStatus
		r.Banned = banned != 0
		r.CreatedAt = createdAt
		r.UpdatedAt = updatedAt
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

// EnrichWithCategories populates the Categories field on the given resources
// by querying the resource_categories join table in bulk.
func (s *ResourceStore) EnrichWithCategories(resources []*Resource) error {
	if len(resources) == 0 {
		return nil
	}
	ids := make([]string, len(resources))
	for i, r := range resources {
		ids[i] = r.ID
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := "SELECT resource_id, category_name FROM resource_categories WHERE resource_id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	catMap := make(map[string][]string)
	for rows.Next() {
		var resID, catName string
		if err := rows.Scan(&resID, &catName); err != nil {
			return err
		}
		catMap[resID] = append(catMap[resID], catName)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range resources {
		if cats, ok := catMap[r.ID]; ok {
			r.Categories = cats
		}
	}
	return nil
}

// ListByAssignedCategoriesPaginated returns resources in categories where the user is assigned,
// plus the user's own uploads. Ordered by creation date descending.
func (s *ResourceStore) ListByAssignedCategoriesPaginated(userID string, limit, offset int) ([]*Resource, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT r.id, r.title, r.filename, r.file_size, r.content_type, r.resource_type,
		       r.views, r.uploaded_by, r.no_transcode,
		       r.transcode_status, r.banned, r.created_at, r.updated_at
		FROM resources r
		LEFT JOIN resource_categories rc ON r.id = rc.resource_id
		LEFT JOIN category_users cu ON rc.category_name = cu.category_name
		WHERE cu.name = ? OR r.uploaded_by = ? OR rc.category_name = 'global'
		ORDER BY r.created_at DESC LIMIT ? OFFSET ?`,
		userID, userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResources(rows)
}

// CountByAssignedCategories returns the count of resources visible to a user
// (in assigned categories + own uploads).
func (s *ResourceStore) CountByAssignedCategories(userID string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(DISTINCT r.id)
		FROM resources r
		LEFT JOIN resource_categories rc ON r.id = rc.resource_id
		LEFT JOIN category_users cu ON rc.category_name = cu.category_name
		WHERE cu.name = ? OR r.uploaded_by = ? OR rc.category_name = 'global'`,
		userID, userID,
	).Scan(&count)
	return count, err
}
