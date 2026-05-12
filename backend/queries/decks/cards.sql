-- name: InsertCard :exec
INSERT INTO decks.cards (id, deck_id, ordinal, term, definition)
VALUES ($1, $2, $3, $4, $5);

-- name: ListCardsByDeck :many
SELECT id, deck_id, ordinal, term, definition
FROM decks.cards
WHERE deck_id = $1
ORDER BY ordinal ASC;

-- name: UpdateCard :exec
UPDATE decks.cards
SET term = $2,
    definition = $3,
    ordinal = $4,
    updated_at = now()
WHERE id = $1;

-- name: DeleteCard :exec
DELETE FROM decks.cards WHERE id = $1;

-- name: DeleteCardsByDeck :exec
DELETE FROM decks.cards WHERE deck_id = $1;
