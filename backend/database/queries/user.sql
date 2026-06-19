-- name: GetUser :one
SELECT id, username, totp_secret, role, created_at FROM users WHERE id = ?;

-- name: GetUserByUsername :one
SELECT id, username, totp_secret, role, created_at FROM users WHERE username = ?;

-- name: ListUsers :many
SELECT id, username, totp_secret, role, created_at FROM users ORDER BY created_at DESC;

-- name: CreateUser :exec
INSERT INTO users (id, username, totp_secret, role) VALUES (?, ?, ?, ?);

-- name: GetAdminUser :one
SELECT id, username, totp_secret, role, created_at FROM users WHERE role = 'admin' LIMIT 1;

-- name: GetAdminUserID :one
SELECT id FROM users WHERE role = 'admin' LIMIT 1;

-- name: CountAdminUsers :one
SELECT COUNT(*) FROM users WHERE role = 'admin';
