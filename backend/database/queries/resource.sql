-- name: GetResource :one
SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE id = ?;

-- name: ListResources :many
SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, banned, no_transcode, created_at, updated_at FROM resources ORDER BY created_at DESC;

-- name: ListResourcesPaginated :many
SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, banned, no_transcode, created_at, updated_at FROM resources ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountResources :one
SELECT COUNT(*) FROM resources;

-- name: ListResourcesByUploader :many
SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE uploaded_by = ? ORDER BY created_at DESC;

-- name: ListResourcesByUploaderPaginated :many
SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE uploaded_by = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountResourcesByUploader :one
SELECT COUNT(*) FROM resources WHERE uploaded_by = ?;

-- name: CreateResource :exec
INSERT INTO resources (id, title, password_hash, filename, file_size, content_type, uploaded_by, category_id, no_transcode) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteResource :exec
DELETE FROM resources WHERE id = ?;

-- name: UpdateTranscodeStatus :exec
UPDATE resources SET transcode_status = ? WHERE id = ?;

-- name: IncrementResourceViews :exec
UPDATE resources SET views = views + 1 WHERE id = ?;

-- name: ListResourcesByTranscodeStatus :many
SELECT id, title, password_hash, filename, file_size, content_type, views, uploaded_by, category_id, transcode_status, banned, no_transcode, created_at, updated_at FROM resources WHERE transcode_status = ? ORDER BY created_at DESC;

-- name: UpdateResourceBanned :exec
UPDATE resources SET banned = ? WHERE id = ?;
