package iam

import "time"

// RevokeReason mirrors the documented reasons in domain-events.md.
type RevokeReason string

const (
	RevokeReasonRotation       RevokeReason = "rotation"
	RevokeReasonLogout         RevokeReason = "logout"
	RevokeReasonPasswordChange RevokeReason = "password-change"
	RevokeReasonReuseDetected  RevokeReason = "reuse-detected"
)

// RefreshToken is one node in a RefreshTokenFamily chain. The opaque value is
// hashed before persistence.
type RefreshToken struct {
	ID         string
	FamilyID   string
	UserID     string
	ParentID   string // empty for the family root
	OpaqueHash string // sha256 hex of the opaque value
	IssuedAt   time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	RevokeNote RevokeReason
}

// IsRevoked reports whether the token has been retired.
func (rt RefreshToken) IsRevoked() bool { return rt.RevokedAt != nil }

// IsExpired reports whether the token is past its TTL relative to now.
func (rt RefreshToken) IsExpired(now time.Time) bool {
	return !rt.ExpiresAt.IsZero() && !now.Before(rt.ExpiresAt)
}

// Revoke retires the token with the given reason. Idempotent: revoking an
// already-revoked token is a no-op.
func (rt *RefreshToken) Revoke(at time.Time, reason RevokeReason) {
	if rt.RevokedAt != nil {
		return
	}
	t := at.UTC()
	rt.RevokedAt = &t
	rt.RevokeNote = reason
}

// RefreshTokenFamily is a non-persisted view used by the rotation use case.
// It contains the chain of tokens belonging to a single login session.
type RefreshTokenFamily struct {
	FamilyID string
	UserID   string
	Tokens   []RefreshToken // sorted by IssuedAt asc
}

// Latest returns the most recently issued token. Empty family returns false.
//
// Invariant: RefreshTokenFamily_TokensAreOrderedByIssuedAt — the slice is kept
// sorted on construction by the repository or by AppendIssued below.
func (f *RefreshTokenFamily) Latest() (RefreshToken, bool) {
	if len(f.Tokens) == 0 {
		return RefreshToken{}, false
	}
	return f.Tokens[len(f.Tokens)-1], true
}

// AppendIssued enforces strict IssuedAt monotonicity (invariant 8).
func (f *RefreshTokenFamily) AppendIssued(t RefreshToken) error {
	if len(f.Tokens) > 0 {
		prev := f.Tokens[len(f.Tokens)-1]
		if !t.IssuedAt.After(prev.IssuedAt) {
			return ErrRefreshTokenInvalid
		}
	}
	f.Tokens = append(f.Tokens, t)
	return nil
}

// HasActiveAfterRevocation reports whether any token in the family is currently
// active (not revoked, not expired). Used by ChangePassword to assert the
// family was indeed retired.
func (f *RefreshTokenFamily) HasActiveAfterRevocation(now time.Time) bool {
	for _, t := range f.Tokens {
		if !t.IsRevoked() && !t.IsExpired(now) {
			return true
		}
	}
	return false
}
