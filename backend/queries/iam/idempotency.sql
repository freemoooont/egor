-- name: InsertIdempotencyKey :exec
INSERT INTO iam.idempotency_keys (
    scope, key, request_hash, response_status, response_body, expires_at
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetIdempotencyKey :one
SELECT scope, key, request_hash, response_status, response_body, created_at, expires_at
FROM iam.idempotency_keys
WHERE scope = $1 AND key = $2;

-- name: DeleteExpiredIdempotencyKeys :exec
DELETE FROM iam.idempotency_keys WHERE expires_at < now();
