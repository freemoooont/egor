// Package iamrepo implements the iam-domain Repositories using pgx/v5 against
// the iam.* schema. Repositories follow the DDD layering rules in
// docs/backend/CLAUDE.md: they accept the active pgx.Tx through context.Context
// (placed there by postgres.UnitOfWork.Do), or fall back to the connection pool
// when no tx is active (read-only flows).
//
// Mapping helpers between sqlc-style row structs and domain entities live in
// mapping.go in this package.
package iamrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domiam "github.com/micocards/api/internal/domain/iam"
	pg "github.com/micocards/api/internal/infrastructure/postgres"
)

// Users is the iam.Users repository, persisting User aggregates.
type Users struct {
	pool *pgxpool.Pool
}

// NewUsers builds a Users repository.
func NewUsers(pool *pgxpool.Pool) *Users { return &Users{pool: pool} }

// Save inserts or updates the given user. Layer 1's RegisterUser flow inserts;
// UpdateProfile and ChangePassword update.
func (r *Users) Save(ctx context.Context, u *domiam.User) error {
	q := pg.Conn(ctx, r.pool)
	const upsert = `
INSERT INTO iam.users (id, email, password_hash, display_name, avatar_kind, avatar_ref, registered_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    display_name = EXCLUDED.display_name,
    avatar_kind = EXCLUDED.avatar_kind,
    avatar_ref = EXCLUDED.avatar_ref,
    updated_at = now()
`
	if _, err := q.Exec(ctx, upsert,
		u.ID(),
		u.Email().String(),
		u.PasswordHash().String(),
		u.DisplayName().String(),
		string(u.Avatar().Kind),
		u.Avatar().Ref,
		u.RegisteredAt(),
	); err != nil {
		if isUniqueViolation(err, "users_email_key") {
			return domiam.ErrEmailTaken
		}
		return fmt.Errorf("iamrepo.Users.Save: %w", err)
	}
	return nil
}

// ByID loads a user by primary key. Returns ErrUserNotFound if missing.
func (r *Users) ByID(ctx context.Context, id string) (*domiam.User, error) {
	row := pg.Conn(ctx, r.pool).QueryRow(ctx, `
SELECT id, email, password_hash, display_name, avatar_kind, avatar_ref, registered_at
FROM iam.users
WHERE id = $1
`, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domiam.ErrUserNotFound
		}
		return nil, fmt.Errorf("iamrepo.Users.ByID: %w", err)
	}
	return u, nil
}

// ByEmail loads a user by email. Returns ErrUserNotFound if missing.
func (r *Users) ByEmail(ctx context.Context, email domiam.EmailAddress) (*domiam.User, error) {
	row := pg.Conn(ctx, r.pool).QueryRow(ctx, `
SELECT id, email, password_hash, display_name, avatar_kind, avatar_ref, registered_at
FROM iam.users
WHERE email = $1
`, email.String())
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domiam.ErrUserNotFound
		}
		return nil, fmt.Errorf("iamrepo.Users.ByEmail: %w", err)
	}
	return u, nil
}

// EmailExists reports whether the email is taken.
func (r *Users) EmailExists(ctx context.Context, email domiam.EmailAddress) (bool, error) {
	var taken bool
	err := pg.Conn(ctx, r.pool).QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM iam.users WHERE email = $1)`,
		email.String(),
	).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("iamrepo.Users.EmailExists: %w", err)
	}
	return taken, nil
}
