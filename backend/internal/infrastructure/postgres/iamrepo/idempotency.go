package iamrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domiam "github.com/micocards/api/internal/domain/iam"
	pg "github.com/micocards/api/internal/infrastructure/postgres"
)

// IdempotencyKeys is the iam.IdempotencyKeys repository (ADR 0005). It lives
// in the iam schema even though it serves every context's middleware (see ADR).
type IdempotencyKeys struct {
	pool *pgxpool.Pool
}

// NewIdempotencyKeys builds the repo.
func NewIdempotencyKeys(pool *pgxpool.Pool) *IdempotencyKeys {
	return &IdempotencyKeys{pool: pool}
}

// Get returns the cached entry, an existence flag, and an error.
func (r *IdempotencyKeys) Get(ctx context.Context, scope, key string) (domiam.IdempotencyEntry, bool, error) {
	row := pg.Conn(ctx, r.pool).QueryRow(ctx, `
SELECT scope, key, request_hash, response_status, response_body, expires_at
FROM iam.idempotency_keys
WHERE scope = $1 AND key = $2
`, scope, key)
	var (
		entry     domiam.IdempotencyEntry
		expiresAt time.Time
	)
	if err := row.Scan(
		&entry.Scope, &entry.Key, &entry.RequestHash,
		&entry.ResponseStatus, &entry.ResponseBody, &expiresAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domiam.IdempotencyEntry{}, false, nil
		}
		return domiam.IdempotencyEntry{}, false, fmt.Errorf("iamrepo.IdempotencyKeys.Get: %w", err)
	}
	entry.ExpiresAtUnix = expiresAt.Unix()
	return entry, true, nil
}

// Put persists the entry. Calls inside the same tx as the originating use case
// (the idempotency middleware opens the tx via UnitOfWork before calling the
// inner handler).
func (r *IdempotencyKeys) Put(ctx context.Context, e domiam.IdempotencyEntry) error {
	expiresAt := time.Unix(e.ExpiresAtUnix, 0).UTC()
	if _, err := pg.Conn(ctx, r.pool).Exec(ctx, `
INSERT INTO iam.idempotency_keys (
    scope, key, request_hash, response_status, response_body, expires_at
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (scope, key) DO UPDATE SET
    request_hash = EXCLUDED.request_hash,
    response_status = EXCLUDED.response_status,
    response_body = EXCLUDED.response_body,
    expires_at = EXCLUDED.expires_at
`, e.Scope, e.Key, e.RequestHash, e.ResponseStatus, e.ResponseBody, expiresAt); err != nil {
		return fmt.Errorf("iamrepo.IdempotencyKeys.Put: %w", err)
	}
	return nil
}
