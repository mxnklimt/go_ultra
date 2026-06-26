-- name: CreateMatch :one
INSERT INTO matches (
    winner_id, loser_id, submitter_id,
    winner_rating_before, loser_rating_before,
    winner_rating_after, loser_rating_after,
    winner_delta, loser_delta,
    played_at, created_at
) VALUES (
    ?, ?, ?,
    ?, ?,
    ?, ?,
    ?, ?,
    ?, ?
)
RETURNING *;

-- name: GetMatchByID :one
SELECT * FROM matches
WHERE id = ?;

-- name: ListGlobalMatches :many
SELECT * FROM matches
WHERE deleted_at IS NULL
ORDER BY played_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: ListPlayerMatches :many
SELECT * FROM matches
WHERE (winner_id = ? OR loser_id = ?) AND deleted_at IS NULL
ORDER BY played_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: GetPlayerHistory :many
SELECT played_at, winner_id, loser_id, winner_rating_after, loser_rating_after
FROM matches
WHERE (winner_id = ? OR loser_id = ?) AND deleted_at IS NULL
ORDER BY played_at ASC, id ASC;

-- name: SoftDeleteMatch :exec
UPDATE matches
SET deleted_at = ?, deleted_by = ?
WHERE id = ? AND deleted_at IS NULL;

-- name: RestoreMatch :exec
UPDATE matches
SET deleted_at = NULL, deleted_by = NULL
WHERE id = ?;

-- name: ListDeletedMatches :many
SELECT * FROM matches
WHERE deleted_at IS NOT NULL
ORDER BY deleted_at DESC;

-- name: CountPlayerWinsLosses :one
SELECT
    COALESCE(SUM(CASE WHEN winner_id = ? THEN 1 ELSE 0 END), 0) AS wins,
    COALESCE(SUM(CASE WHEN loser_id  = ? THEN 1 ELSE 0 END), 0) AS losses
FROM matches
WHERE (winner_id = ? OR loser_id = ?) AND deleted_at IS NULL;
