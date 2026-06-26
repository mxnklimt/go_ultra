-- name: CreatePlayer :one
INSERT INTO players (username, rating)
VALUES (?, ?)
RETURNING *;

-- name: GetPlayerByID :one
SELECT * FROM players
WHERE id = ?;

-- name: GetPlayerByUsername :one
SELECT * FROM players
WHERE username = ?;

-- name: ListPlayersByRating :many
SELECT * FROM players
ORDER BY rating DESC, id ASC;

-- name: UpdatePlayerRating :exec
UPDATE players
SET rating = ?
WHERE id = ?;
