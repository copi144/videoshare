-- name: CreateShareResource :exec
INSERT INTO share_resources (resource_id, password, expires_at, created_by, created_at) VALUES (?, ?, ?, ?, ?);

-- name: GetShareResource :one
SELECT resource_id, password, expires_at, created_by, created_at FROM share_resources WHERE resource_id = ? AND password = ? AND (expires_at IS NULL OR expires_at > ?);

-- name: ListShareResources :many
SELECT resource_id, password, expires_at, created_by, created_at FROM share_resources WHERE resource_id = ? AND (expires_at IS NULL OR expires_at > ?) ORDER BY created_at DESC;

-- name: DeleteShareResource :exec
DELETE FROM share_resources WHERE resource_id = ? AND password = ?;

-- name: DeleteExpiredShareResources :exec
DELETE FROM share_resources WHERE expires_at IS NOT NULL AND expires_at <= ?;
