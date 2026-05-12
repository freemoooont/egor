-- name: InsertSession :exec
INSERT INTO practice.sessions (
    id, user_id, deck_id, mode, status, card_ids, started_at, completed_at, abandoned_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetSessionByID :one
SELECT id, user_id, deck_id, mode, status, card_ids, started_at, completed_at, abandoned_at
FROM practice.sessions
WHERE id = $1;

-- name: UpdateSession :exec
UPDATE practice.sessions
SET status = $2,
    completed_at = $3,
    abandoned_at = $4
WHERE id = $1;

-- name: GetLatestCompletedSession :one
SELECT id, user_id, deck_id, mode, status, card_ids, started_at, completed_at, abandoned_at
FROM practice.sessions
WHERE user_id = $1 AND deck_id = $2 AND completed_at IS NOT NULL
ORDER BY completed_at DESC
LIMIT 1;
