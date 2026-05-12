// Package auth implements the iam application-layer ports: PasswordHasher
// (bcrypt), AccessTokenSigner (JWT HS256), and RefreshTokenMinter
// (crypto/rand + sha256).
package auth

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	domiam "github.com/micocards/api/internal/domain/iam"
)

// MinPasswordLength is the soft lower bound enforced by Strength. Configurable
// via env in main; default 8.
const MinPasswordLength = 8

// BcryptHasher implements the iam.PasswordHasher port using golang.org/x/crypto/bcrypt.
type BcryptHasher struct {
	cost      int
	minLength int
}

// NewBcryptHasher builds a hasher. cost is clamped to bcrypt.MinCost..bcrypt.MaxCost.
func NewBcryptHasher(cost, minLength int) *BcryptHasher {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	if cost > bcrypt.MaxCost {
		cost = bcrypt.MaxCost
	}
	if minLength <= 0 {
		minLength = MinPasswordLength
	}
	return &BcryptHasher{cost: cost, minLength: minLength}
}

// Hash returns a bcrypt hash wrapped in the domain's PasswordHash VO.
func (h *BcryptHasher) Hash(_ context.Context, plaintext string) (domiam.PasswordHash, error) {
	if err := h.Strength(plaintext); err != nil {
		return domiam.PasswordHash{}, err
	}
	raw, err := bcrypt.GenerateFromPassword([]byte(plaintext), h.cost)
	if err != nil {
		return domiam.PasswordHash{}, err
	}
	return domiam.NewPasswordHash(string(raw))
}

// Compare verifies the plaintext against the stored hash. Returns
// ErrInvalidCredentials on mismatch.
func (h *BcryptHasher) Compare(_ context.Context, hash domiam.PasswordHash, plaintext string) error {
	if hash.IsZero() {
		return domiam.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash.String()), []byte(plaintext)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return domiam.ErrInvalidCredentials
		}
		return err
	}
	return nil
}

// Strength enforces the minimum-length rule. Domain semantics: too-short
// passwords return ErrPasswordTooWeak.
func (h *BcryptHasher) Strength(plaintext string) error {
	if len(plaintext) < h.minLength {
		return domiam.ErrPasswordTooWeak
	}
	return nil
}
