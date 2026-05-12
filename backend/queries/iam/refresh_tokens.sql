-- name: InsertRefreshToken :exec
INSERT INTO iam.refresh_tokens (
    id, family_id, user_id, parent_id, opaque_hash,
    issued_at, expires_at, revoked_at, revoke_note
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetRefreshTokenByHash :one
SELECT id, family_id, user_id, parent_id, opaque_hash,
       issued_at, expires_at, revoked_at, revoke_note
FROM iam.refresh_tokens
WHERE opaque_hash = $1;

-- name: GetRefreshTokensByFamily :many
SELECT id, family_id, user_id, parent_id, opaque_hash,
       issued_at, expires_at, revoked_at, revoke_note
FROM iam.refresh_tokens
WHERE family_id = $1
ORDER BY issued_at ASC;

-- name: RevokeRefreshTokenByID :exec
UPDATE iam.refresh_tokens
SET revoked_at = $2, revoke_note = $3
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeRefreshFamily :exec
UPDATE iam.refresh_tokens
SET revoked_at = $2, revoke_note = $3
WHERE family_id = $1 AND revoked_at IS NULL;

-- name: RevokeRefreshFamilyForUser :exec
UPDATE iam.refresh_tokens
SET revoked_at = $2, revoke_note = $3
WHERE user_id = $1 AND revoked_at IS NULL;
