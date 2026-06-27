-- name: CreateSession :exec
INSERT INTO sessions (token, player_id, created_at, expires_at)
VALUES (?, ?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions
WHERE token = ?;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= ?;

-- name: CreateAdminSession :exec
INSERT INTO admin_sessions (token, created_at, expires_at)
VALUES (?, ?, ?);

-- name: GetAdminSession :one
SELECT * FROM admin_sessions
WHERE token = ?;

-- name: DeleteAdminSession :exec
DELETE FROM admin_sessions
WHERE token = ?;
