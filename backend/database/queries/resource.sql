-- name: GetResource :one
SELECT id, title, filename, file_size, content_type, resource_type, views, uploaded_by, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE id = ?;

-- name: ListResources :many
SELECT id, title, filename, file_size, content_type, resource_type, views, uploaded_by, transcode_status, banned, no_transcode, created_at, updated_at FROM resources ORDER BY created_at DESC;

-- name: ListResourcesPaginated :many
SELECT id, title, filename, file_size, content_type, resource_type, views, uploaded_by, transcode_status, banned, no_transcode, created_at, updated_at FROM resources ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountResources :one
SELECT COUNT(*) FROM resources;

-- name: ListResourcesByUploader :many
SELECT id, title, filename, file_size, content_type, resource_type, views, uploaded_by, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE uploaded_by = ? ORDER BY created_at DESC;

-- name: ListResourcesByUploaderPaginated :many
SELECT id, title, filename, file_size, content_type, resource_type, views, uploaded_by, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE uploaded_by = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountResourcesByUploader :one
SELECT COUNT(*) FROM resources WHERE uploaded_by = ?;

-- name: CreateResource :exec
INSERT INTO resources (id, title, filename, file_size, content_type, resource_type, uploaded_by, no_transcode) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteResource :exec
DELETE FROM resources WHERE id = ?;

-- name: UpdateTranscodeStatus :exec
UPDATE resources SET transcode_status = ? WHERE id = ?;

-- name: IncrementResourceViews :exec
UPDATE resources SET views = views + 1 WHERE id = ?;

-- name: ListResourcesByTranscodeStatus :many
SELECT id, title, filename, file_size, content_type, resource_type, views, uploaded_by, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE transcode_status = ? ORDER BY created_at DESC;

-- name: UpdateResourceBanned :exec
UPDATE resources SET banned = ? WHERE id = ?;

-- name: ListResourcesByTypePaginated :many
SELECT id, title, filename, file_size, content_type, resource_type, views, uploaded_by, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE resource_type = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountResourcesByType :one
SELECT COUNT(*) FROM resources WHERE resource_type = ?;

-- name: GetResourceCategoryCount :one
SELECT COUNT(*) FROM resource_categories WHERE resource_id = ?;

-- name: AddResourceCategory :exec
INSERT INTO resource_categories (resource_id, category_name) VALUES (?, ?);

-- name: RemoveResourceFromCategory :exec
DELETE FROM resource_categories WHERE resource_id = ? AND category_name = ?;

-- name: RemoveAllResourceCategories :exec
DELETE FROM resource_categories WHERE resource_id = ?;

-- name: ListResourceCategories :many
SELECT category_name FROM resource_categories WHERE resource_id = ?;

-- name: ListResourcesByCategoryPaginated :many
SELECT r.id, r.title, r.filename, r.file_size, r.content_type, r.resource_type, r.views, r.uploaded_by, r.transcode_status, r.banned, r.no_transcode, r.created_at, r.updated_at FROM resources r JOIN resource_categories rc ON r.id = rc.resource_id WHERE rc.category_name = ? ORDER BY r.created_at DESC LIMIT ? OFFSET ?;

-- name: CountResourcesByCategory :one
SELECT COUNT(*) FROM resource_categories WHERE category_name = ?;
