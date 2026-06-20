-- name: GetPlaylist :one
SELECT id, category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE id = ?;

-- name: ListPlaylistsByCategory :many
SELECT id, category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE category_name = ? ORDER BY sort_order ASC, created_at ASC;

-- name: ListAllPlaylists :many
SELECT id, category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists ORDER BY sort_order ASC, created_at ASC;

-- name: ListPlaylistsPaginated :many
SELECT id, category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists ORDER BY sort_order ASC, created_at ASC LIMIT ? OFFSET ?;

-- name: CountPlaylists :one
SELECT COUNT(*) FROM playlists;

-- name: CreatePlaylist :exec
INSERT INTO playlists (id, category_name, playlist_type, name, display_name, description, created_by, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePlaylist :exec
UPDATE playlists SET name = ?, display_name = ?, description = ?, sort_order = ? WHERE id = ?;

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

-- name: ListPlaylistsByType :many
SELECT id, category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE playlist_type = ? ORDER BY sort_order ASC, created_at ASC;

-- name: ListPlaylistsByTypePaginated :many
SELECT id, category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE playlist_type = ? ORDER BY sort_order ASC, created_at ASC LIMIT ? OFFSET ?;

-- name: CountPlaylistsByType :one
SELECT COUNT(*) FROM playlists WHERE playlist_type = ?;

-- name: ListPlaylistsByCategoryAndType :many
SELECT id, category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE category_name = ? AND playlist_type = ? ORDER BY sort_order ASC, created_at ASC;
