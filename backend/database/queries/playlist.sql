-- name: GetPlaylist :one
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE category_name = ? AND name = ?;

-- name: ListPlaylistsByCategory :many
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE category_name = ? ORDER BY sort_order ASC, created_at ASC;

-- name: ListAllPlaylists :many
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists ORDER BY sort_order ASC, created_at ASC;

-- name: ListPlaylistsPaginated :many
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists ORDER BY sort_order ASC, created_at ASC LIMIT ? OFFSET ?;

-- name: CountPlaylists :one
SELECT COUNT(*) FROM playlists;

-- name: CreatePlaylist :exec
INSERT INTO playlists (category_name, playlist_type, name, display_name, description, created_by, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePlaylist :exec
UPDATE playlists SET display_name = ?, description = ?, sort_order = ? WHERE category_name = ? AND name = ?;

-- name: DeletePlaylist :exec
DELETE FROM playlists WHERE category_name = ? AND name = ?;

-- name: AddVideoToPlaylist :exec
INSERT INTO playlist_videos (playlist_category_name, playlist_name, resource_id, sort_order) VALUES (?, ?, ?, ?) ON CONFLICT(playlist_category_name, playlist_name, resource_id) DO UPDATE SET sort_order = EXCLUDED.sort_order;

-- name: RemoveVideoFromPlaylist :exec
DELETE FROM playlist_videos WHERE playlist_category_name = ? AND playlist_name = ? AND resource_id = ?;

-- name: ListPlaylistVideos :many
SELECT resource_id FROM playlist_videos WHERE playlist_category_name = ? AND playlist_name = ? ORDER BY sort_order ASC;

-- name: GetPlaylistsForResource :many
SELECT playlist_category_name, playlist_name FROM playlist_videos WHERE resource_id = ?;

-- name: ListPlaylistsByType :many
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE playlist_type = ? ORDER BY sort_order ASC, created_at ASC;

-- name: ListPlaylistsByTypePaginated :many
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE playlist_type = ? ORDER BY sort_order ASC, created_at ASC LIMIT ? OFFSET ?;

-- name: CountPlaylistsByType :one
SELECT COUNT(*) FROM playlists WHERE playlist_type = ?;

-- name: ListPlaylistsByCategoryAndType :many
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE category_name = ? AND playlist_type = ? ORDER BY sort_order ASC, created_at ASC;

-- name: ListPlaylistsByCategoryPaginated :many
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE category_name = ? ORDER BY sort_order ASC, created_at ASC LIMIT ? OFFSET ?;

-- name: CountPlaylistsByCategory :one
SELECT COUNT(*) FROM playlists WHERE category_name = ?;

-- name: ListPlaylistsByCategoryAndTypePaginated :many
SELECT category_name, playlist_type, name, display_name, description, created_by, sort_order, created_at FROM playlists WHERE category_name = ? AND playlist_type = ? ORDER BY sort_order ASC, created_at ASC LIMIT ? OFFSET ?;

-- name: CountPlaylistsByCategoryAndType :one
SELECT COUNT(*) FROM playlists WHERE category_name = ? AND playlist_type = ?;
