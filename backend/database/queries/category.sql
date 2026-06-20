-- name: GetCategory :one
SELECT name, display_name, description, created_by, created_at FROM categories WHERE name = ?;

-- name: ListCategories :many
SELECT name, display_name, description, created_by, created_at FROM categories ORDER BY created_at DESC;

-- name: ListCategoriesPaginated :many
SELECT name, display_name, description, created_by, created_at FROM categories ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories;

-- name: ListCategoriesByUploader :many
SELECT c.name, c.display_name, c.description, c.created_by, c.created_at FROM categories c JOIN category_users cu ON cu.category_name = c.name WHERE cu.name = ? AND cu.can_upload = 1 ORDER BY c.created_at DESC;

-- name: ListCategoriesByUploaderPaginated :many
SELECT c.name, c.display_name, c.description, c.created_by, c.created_at FROM categories c JOIN category_users cu ON cu.category_name = c.name WHERE cu.name = ? AND cu.can_upload = 1 ORDER BY c.created_at DESC LIMIT ? OFFSET ?;

-- name: CountCategoriesByUploader :one
SELECT COUNT(*) FROM categories c JOIN category_users cu ON cu.category_name = c.name WHERE cu.name = ? AND cu.can_upload = 1;

-- name: CreateCategory :exec
INSERT INTO categories (name, display_name, description, created_by) VALUES (?, ?, ?, ?);

-- name: DeleteCategory :exec
DELETE FROM categories WHERE name = ?;

-- name: GetCategoryVideoCount :one
SELECT COUNT(*) FROM resource_categories WHERE category_name = ?;

-- name: ListUploaders :many
SELECT name, can_upload FROM category_users WHERE category_name = ?;

-- name: IsAssigned :one
SELECT COUNT(*) FROM category_users WHERE category_name = ? AND name = ?;

-- name: CanUpload :one
SELECT COUNT(*) FROM category_users WHERE category_name = ? AND name = ? AND can_upload = 1;

-- name: ClearCategoryUploaders :exec
DELETE FROM category_users WHERE category_name = ?;

-- name: AddUploader :exec
INSERT INTO category_users (category_name, name, can_upload) VALUES (?, ?, ?);

-- name: RemoveUploader :exec
DELETE FROM category_users WHERE category_name = ? AND name = ?;
