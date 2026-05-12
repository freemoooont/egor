package iam

import "context"

// Users is the persistence port for the User aggregate. Implementations live in
// internal/infrastructure/postgres/iamrepo (layer 2).
type Users interface {
	Save(ctx context.Context, u *User) error
	ByID(ctx context.Context, id string) (*User, error)
	ByEmail(ctx context.Context, email EmailAddress) (*User, error)
	EmailExists(ctx context.Context, email EmailAddress) (bool, error)
}

// RefreshTokens persists rotating refresh tokens.
type RefreshTokens interface {
	Save(ctx context.Context, t RefreshToken) error
	ByOpaqueHash(ctx context.Context, hash string) (RefreshToken, error)
	FamilyByID(ctx context.Context, familyID string) (RefreshTokenFamily, error)
	RevokeFamily(ctx context.Context, familyID string, at TimeFn, reason RevokeReason) error
	RevokeAllForUser(ctx context.Context, userID string, at TimeFn, reason RevokeReason) error
	RevokeOne(ctx context.Context, tokenID string, at TimeFn, reason RevokeReason) error
}

// TimeFn lets the repository fix a single "now" per tx without dragging the
// clock into the domain interface.
type TimeFn = func() (year, month, day int)

// IdempotencyKeys is the cross-cutting persistence port (ADR 0005). It lives in
// the iam context because that's where the table is owned, even though the
// middleware uses it from every context.
type IdempotencyKeys interface {
	Get(ctx context.Context, scope, key string) (IdempotencyEntry, bool, error)
	Put(ctx context.Context, e IdempotencyEntry) error
}

// IdempotencyEntry mirrors the cached response row.
type IdempotencyEntry struct {
	Scope          string
	Key            string
	RequestHash    string
	ResponseStatus int
	ResponseBody   []byte
	ExpiresAtUnix  int64
}
