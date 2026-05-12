-- name: UpsertUserDeckProgress :exec
INSERT INTO practice.user_deck_progress (
    user_id, deck_id, cards, know_count, learning_count, dont_know_count, last_practiced_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (user_id, deck_id)
DO UPDATE SET cards = EXCLUDED.cards,
              know_count = EXCLUDED.know_count,
              learning_count = EXCLUDED.learning_count,
              dont_know_count = EXCLUDED.dont_know_count,
              last_practiced_at = EXCLUDED.last_practiced_at,
              updated_at = now();

-- name: GetUserDeckProgress :one
SELECT user_id, deck_id, cards, know_count, learning_count, dont_know_count, last_practiced_at, updated_at
FROM practice.user_deck_progress
WHERE user_id = $1 AND deck_id = $2;

-- name: DeleteProgressByDeck :exec
DELETE FROM practice.user_deck_progress WHERE deck_id = $1;
