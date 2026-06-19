-- name: GetCategory :one
SELECT id, name, description, created_by, created_at FROM categories WHERE id = ?;

-- name: ListCategories :many
SELECT id, name, description, created_by, created_at FROM categories ORDER BY created_at DESC;

-- name: ListCategoriesPaginated :many
SELECT id, name, description, created_by, created_at FROM categories ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories;

-- name: ListCategoriesByUploader :many
SELECT c.id, c.name, c.description, c.created_by, c.created_at FROM categories c JOIN category_uploaders cu ON cu.category_id = c.id WHERE cu.user_id = ? ORDER BY c.created_at DESC;

-- name: ListCategoriesByUploaderPaginated :many
SELECT c.id, c.name, c.description, c.created_by, c.created_at FROM categories c JOIN category_uploaders cu ON cu.category_id = c.id WHERE cu.user_id = ? ORDER BY c.created_at DESC LIMIT ? OFFSET ?;

-- name: CountCategoriesByUploader :one
SELECT COUNT(*) FROM categories c JOIN category_uploaders cu ON cu.category_id = c.id WHERE cu.user_id = ?;

-- name: CreateCategory :exec
INSERT INTO categories (id, name, description, created_by) VALUES (?, ?, ?, ?);

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = ?;

-- name: GetCategoryVideoCount :one
SELECT COUNT(*) FROM resources WHERE category_id = ?;

-- name: GetCategoryUploaders :many
SELECT user_id FROM category_uploaders WHERE category_id = ?;

-- name: IsUploaderAuthorized :one
SELECT COUNT(*) FROM category_uploaders WHERE category_id = ? AND user_id = ?;

-- name: ClearCategoryUploaders :exec
DELETE FROM category_uploaders WHERE category_id = ?;

-- name: AddCategoryUploader :exec
INSERT INTO category_uploaders (category_id, user_id) VALUES (?, ?);
