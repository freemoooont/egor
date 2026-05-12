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

// RefreshTokens is the iam.RefreshTokens repository.
type RefreshTokens struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

// NewRefreshTokens builds a RefreshTokens repository. now is used to stamp the
// revoked_at timestamps when the domain TimeFn is nil (the application
// currently passes nil — see ports.go).
func NewRefreshTokens(pool *pgxpool.Pool, now func() time.Time) *RefreshTokens {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &RefreshTokens{pool: pool, now: now}
}

// Save inserts a new refresh-token row. Updates are performed via the explicit
// Revoke* methods below, never via Save.
func (r *RefreshTokens) Save(ctx context.Context, t domiam.RefreshToken) error {
	q := pg.Conn(ctx, r.pool)
	const ins = `
INSERT INTO iam.refresh_tokens (
    id, family_id, user_id, parent_id, opaque_hash, issued_at, expires_at, revoked_at, revoke_note
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
    revoked_at = EXCLUDED.revoked_at,
    revoke_note = EXCLUDED.revoke_note
`
	if _, err := q.Exec(ctx, ins,
		t.ID, t.FamilyID, t.UserID, t.ParentID, t.OpaqueHash,
		t.IssuedAt, t.ExpiresAt, t.RevokedAt, string(t.RevokeNote),
	); err != nil {
		return fmt.Errorf("iamrepo.RefreshTokens.Save: %w", err)
	}
	return nil
}

// ByOpaqueHash loads a refresh token by its sha256 hash. Returns
// ErrRefreshTokenInvalid when missing.
func (r *RefreshTokens) ByOpaqueHash(ctx context.Context, hash string) (domiam.RefreshToken, error) {
	row := pg.Conn(ctx, r.pool).QueryRow(ctx, `
SELECT id, family_id, user_id, parent_id, opaque_hash, issued_at, expires_at, revoked_at, revoke_note
FROM iam.refresh_tokens
WHERE opaque_hash = $1
`, hash)
	t, err := scanRefreshToken(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domiam.RefreshToken{}, domiam.ErrRefreshTokenInvalid
		}
		return domiam.RefreshToken{}, fmt.Errorf("iamrepo.RefreshTokens.ByOpaqueHash: %w", err)
	}
	return t, nil
}

// FamilyByID loads all tokens belonging to the supplied family, sorted by
// issued_at asc (matches the AppendIssued invariant).
func (r *RefreshTokens) FamilyByID(ctx context.Context, familyID string) (domiam.RefreshTokenFamily, error) {
	rows, err := pg.Conn(ctx, r.pool).Query(ctx, `
SELECT id, family_id, user_id, parent_id, opaque_hash, issued_at, expires_at, revoked_at, revoke_note
FROM iam.refresh_tokens
WHERE family_id = $1
ORDER BY issued_at ASC, id ASC
`, familyID)
	if err != nil {
		return domiam.RefreshTokenFamily{}, fmt.Errorf("iamrepo.RefreshTokens.FamilyByID: %w", err)
	}
	defer rows.Close()
	fam := domiam.RefreshTokenFamily{FamilyID: familyID}
	for rows.Next() {
		t, err := scanRefreshToken(rows)
		if err != nil {
			return domiam.RefreshTokenFamily{}, fmt.Errorf("iamrepo.RefreshTokens.FamilyByID scan: %w", err)
		}
		fam.UserID = t.UserID
		fam.Tokens = append(fam.Tokens, t)
	}
	if err := rows.Err(); err != nil {
		return domiam.RefreshTokenFamily{}, err
	}
	return fam, nil
}

// RevokeFamily marks every active token in the family revoked.
func (r *RefreshTokens) RevokeFamily(ctx context.Context, familyID string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	now := r.now()
	if _, err := pg.Conn(ctx, r.pool).Exec(ctx, `
UPDATE iam.refresh_tokens
SET revoked_at = $2, revoke_note = $3
WHERE family_id = $1 AND revoked_at IS NULL
`, familyID, now, string(reason)); err != nil {
		return fmt.Errorf("iamrepo.RefreshTokens.RevokeFamily: %w", err)
	}
	return nil
}

// RevokeAllForUser revokes every active token belonging to the given user.
func (r *RefreshTokens) RevokeAllForUser(ctx context.Context, userID string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	now := r.now()
	if _, err := pg.Conn(ctx, r.pool).Exec(ctx, `
UPDATE iam.refresh_tokens
SET revoked_at = $2, revoke_note = $3
WHERE user_id = $1 AND revoked_at IS NULL
`, userID, now, string(reason)); err != nil {
		return fmt.Errorf("iamrepo.RefreshTokens.RevokeAllForUser: %w", err)
	}
	return nil
}

// RevokeOne revokes a single token. Idempotent — a no-op on already-revoked
// rows.
func (r *RefreshTokens) RevokeOne(ctx context.Context, tokenID string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	now := r.now()
	tag, err := pg.Conn(ctx, r.pool).Exec(ctx, `
UPDATE iam.refresh_tokens
SET revoked_at = $2, revoke_note = $3
WHERE id = $1 AND revoked_at IS NULL
`, tokenID, now, string(reason))
	if err != nil {
		return fmt.Errorf("iamrepo.RefreshTokens.RevokeOne: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Either already revoked or missing. Match the in-memory fake's
		// "missing => ErrRefreshTokenInvalid; already revoked => no-op"
		// semantics by checking existence.
		var exists bool
		if err := pg.Conn(ctx, r.pool).QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM iam.refresh_tokens WHERE id = $1)`, tokenID,
		).Scan(&exists); err != nil {
			return fmt.Errorf("iamrepo.RefreshTokens.RevokeOne exists check: %w", err)
		}
		if !exists {
			return domiam.ErrRefreshTokenInvalid
		}
	}
	return nil
}
