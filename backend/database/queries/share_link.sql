-- name: CreateShareLink :exec
INSERT INTO share_links (id, password, target_type, target_id, expires_at, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetShareLink :one
SELECT id, password, target_type, target_id, expires_at, created_by, created_at FROM share_links WHERE id = ? AND (expires_at IS NULL OR expires_at > ?);

-- name: ListShareLinksByTarget :many
SELECT id, password, target_type, target_id, expires_at, created_by, created_at FROM share_links WHERE target_type = ? AND target_id = ? AND (expires_at IS NULL OR expires_at > ?) ORDER BY created_at DESC;

-- name: DeleteShareLink :exec
DELETE FROM share_links WHERE id = ?;

-- name: DeleteExpiredShareLinks :exec
DELETE FROM share_links WHERE expires_at IS NOT NULL AND expires_at <= ?;
