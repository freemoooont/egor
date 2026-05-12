-- name: UpsertRating :exec
INSERT INTO practice.session_card_ratings (id, session_id, card_id, rating, rated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (session_id, card_id)
DO UPDATE SET rating = EXCLUDED.rating, rated_at = EXCLUDED.rated_at;

-- name: ListRatingsBySession :many
SELECT id, session_id, card_id, rating, rated_at
FROM practice.session_card_ratings
WHERE session_id = $1
ORDER BY rated_at ASC;

-- name: DeleteRatingsBySession :exec
DELETE FROM practice.session_card_ratings WHERE session_id = $1;
