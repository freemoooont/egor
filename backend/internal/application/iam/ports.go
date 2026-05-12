// Package iam contains the iam-context use cases. It depends only on the iam
// domain plus shared/clock/idgen ports. No infra, no HTTP, no DB.
package iam

import (
	"context"

	"github.com/micocards/api/internal/domain/iam"
)

// PasswordHasher converts plaintext into a bcrypt PasswordHash and verifies
// candidates against stored hashes. Implementations live in
// internal/infrastructure/auth.
type PasswordHasher interface {
	Hash(ctx context.Context, plaintext string) (iam.PasswordHash, error)
	Compare(ctx context.Context, hash iam.PasswordHash, plaintext string) error
	Strength(plaintext string) error // returns ErrPasswordTooWeak when too short etc.
}

// AccessTokenSigner produces short-lived JWT access tokens.
type AccessTokenSigner interface {
	SignAccessToken(ctx context.Context, userID string) (token string, expiresAtUnix int64, err error)
	VerifyAccessToken(ctx context.Context, token string) (userID string, err error)
}

// RefreshTokenMinter mints opaque refresh tokens. Returned plaintext is shown
// to the client exactly once; the hash is what gets persisted.
type RefreshTokenMinter interface {
	Mint(ctx context.Context) (plaintext, hash string, err error)
}

// EventPublisher dispatches domain events on the synchronous in-process bus.
// Layer 2 will pair this with the outbox writer (ADR 0002).
type EventPublisher interface {
	Publish(ctx context.Context, events ...iam.Event) error
}

// Outbox is the persistence port for the per-context outbox table (ADR 0002).
// Layer 2 implements it; layer 1 only wires the contract.
type Outbox interface {
	Append(ctx context.Context, eventName string, payload []byte, idempotencyKey string) error
}

// UnitOfWork runs the supplied callback inside a single pgx.Tx. Layer 2
// implements it; layer 1 has an in-memory fake for tests.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// AccessTokenTTL is the canonical lifetime of access tokens (ADR 0003).
const AccessTokenTTLSeconds = 15 * 60

// RefreshTokenTTL is the canonical lifetime of refresh tokens (ADR 0003).
const RefreshTokenTTLSeconds = 7 * 24 * 60 * 60
