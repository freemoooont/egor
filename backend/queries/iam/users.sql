-- name: InsertUser :exec
INSERT INTO iam.users (id, email, password_hash, display_name, avatar_kind, avatar_ref, registered_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetUserByID :one
SELECT id, email, password_hash, display_name, avatar_kind, avatar_ref, registered_at
FROM iam.users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, display_name, avatar_kind, avatar_ref, registered_at
FROM iam.users
WHERE email = $1;

-- name: EmailExists :one
SELECT EXISTS (SELECT 1 FROM iam.users WHERE email = $1) AS taken;

-- name: UpdateUserProfile :exec
UPDATE iam.users
SET email = $2,
    display_name = $3,
    avatar_kind = $4,
    avatar_ref = $5,
    updated_at = now()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE iam.users
SET password_hash = $2,
    updated_at = now()
WHERE id = $1;
