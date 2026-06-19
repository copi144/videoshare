-- name: GetPlaylist :one
SELECT id, category_id, name, description, created_by, sort_order, created_at FROM playlists WHERE id = ?;

-- name: ListPlaylistsByCategory :many
SELECT id, category_id, name, description, created_by, sort_order, created_at FROM playlists WHERE category_id = ? ORDER BY sort_order ASC, created_at ASC;

-- name: ListAllPlaylists :many
SELECT id, category_id, name, description, created_by, sort_order, created_at FROM playlists ORDER BY sort_order ASC, created_at ASC;

-- name: ListPlaylistsPaginated :many
SELECT id, category_id, name, description, created_by, sort_order, created_at FROM playlists ORDER BY sort_order ASC, created_at ASC LIMIT ? OFFSET ?;

-- name: CountPlaylists :one
SELECT COUNT(*) FROM playlists;

-- name: CreatePlaylist :exec
INSERT INTO playlists (id, category_id, name, description, created_by, sort_order) VALUES (?, ?, ?, ?, ?, ?);

-- name: DeletePlaylist :exec
DELETE FROM playlists WHERE id = ?;

-- name: AddVideoToPlaylist :exec
INSERT INTO playlist_videos (playlist_id, resource_id, sort_order) VALUES (?, ?, ?) ON CONFLICT(playlist_id, resource_id) DO UPDATE SET sort_order = EXCLUDED.sort_order;

-- name: RemoveVideoFromPlaylist :exec
DELETE FROM playlist_videos WHERE playlist_id = ? AND resource_id = ?;

-- name: ListPlaylistVideos :many
SELECT resource_id FROM playlist_videos WHERE playlist_id = ? ORDER BY sort_order ASC;

-- name: GetPlaylistsForResource :many
SELECT playlist_id FROM playlist_videos WHERE resource_id = ?;
