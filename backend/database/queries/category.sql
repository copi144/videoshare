-- name: GetCategory :one
SELECT name, display_name, description, created_by, created_at FROM categories WHERE name = ?;

-- name: ListCategories :many
SELECT name, display_name, description, created_by, created_at FROM categories ORDER BY created_at DESC;

-- name: ListCategoriesPaginated :many
SELECT name, display_name, description, created_by, created_at FROM categories ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories;

-- name: ListCategoriesByUploader :many
SELECT c.name, c.display_name, c.description, c.created_by, c.created_at FROM categories c JOIN category_users cu ON cu.category_name = c.name WHERE cu.user_id = ? ORDER BY c.created_at DESC;

-- name: ListCategoriesByUploaderPaginated :many
SELECT c.name, c.display_name, c.description, c.created_by, c.created_at FROM categories c JOIN category_users cu ON cu.category_name = c.name WHERE cu.user_id = ? ORDER BY c.created_at DESC LIMIT ? OFFSET ?;

-- name: CountCategoriesByUploader :one
SELECT COUNT(*) FROM categories c JOIN category_users cu ON cu.category_name = c.name WHERE cu.user_id = ?;

-- name: CreateCategory :exec
INSERT INTO categories (name, display_name, description, created_by) VALUES (?, ?, ?, ?);

-- name: DeleteCategory :exec
DELETE FROM categories WHERE name = ?;

-- name: GetCategoryVideoCount :one
SELECT COUNT(*) FROM resources WHERE category_name = ?;

-- name: ListUploaders :many
SELECT user_id FROM category_users WHERE category_name = ?;

-- name: IsUploaderAuthorized :one
SELECT COUNT(*) FROM category_users WHERE category_name = ? AND user_id = ?;

-- name: ClearCategoryUploaders :exec
DELETE FROM category_users WHERE category_name = ?;

-- name: AddUploader :exec
INSERT INTO category_users (category_name, user_id) VALUES (?, ?);

-- name: RemoveUploader :exec
DELETE FROM category_users WHERE category_name = ? AND user_id = ?;
