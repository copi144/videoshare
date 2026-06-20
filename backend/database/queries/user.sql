-- name: GetUser :one
SELECT name, totp_secret, display_name, is_admin, created_at FROM users WHERE name = ?;

-- name: ListUsers :many
SELECT name, totp_secret, display_name, is_admin, created_at FROM users ORDER BY created_at DESC;

-- name: CreateUser :exec
INSERT INTO users (name, totp_secret, display_name, is_admin) VALUES (?, ?, ?, ?);

-- name: GetAdminUser :one
SELECT name, totp_secret, display_name, is_admin, created_at FROM users WHERE is_admin = 1 LIMIT 1;

-- name: GetAdminName :one
SELECT name FROM users WHERE is_admin = 1 LIMIT 1;

-- name: CountAdminUsers :one
SELECT COUNT(*) FROM users WHERE is_admin = 1;
