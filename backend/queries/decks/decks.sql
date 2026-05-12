-- name: InsertDeck :exec
INSERT INTO decks.decks (id, owner_id, title, created_at, deleted_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetDeckByID :one
SELECT id, owner_id, title, created_at, deleted_at
FROM decks.decks
WHERE id = $1;

-- name: ListDecksByOwner :many
SELECT id, owner_id, title, created_at, deleted_at
FROM decks.decks
WHERE owner_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: UpdateDeck :exec
UPDATE decks.decks
SET title = $2,
    deleted_at = $3,
    updated_at = now()
WHERE id = $1;

-- name: SoftDeleteDeck :exec
UPDATE decks.decks
SET deleted_at = $2,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
