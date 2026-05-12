package iam

import "strings"

// PasswordHash is an opaque, already-hashed credential. Plaintext never crosses
// this boundary. Implementations of the application-layer Hasher port produce
// these.
type PasswordHash struct {
	value string
}

// NewPasswordHash wraps an already-hashed value. We accept any non-empty
// string that LOOKS hashed (bcrypt: $2a$ / $2b$ / $2y$ prefix). Plaintext
// strings without a recognisable hash prefix are rejected.
//
// Invariant: User_PasswordHashMustComeFromBcryptHasher.
func NewPasswordHash(hashed string) (PasswordHash, error) {
	if strings.TrimSpace(hashed) == "" {
		return PasswordHash{}, ErrInvalidPasswordHash
	}
	if !looksHashed(hashed) {
		return PasswordHash{}, ErrInvalidPasswordHash
	}
	return PasswordHash{value: hashed}, nil
}

func looksHashed(v string) bool {
	switch {
	case strings.HasPrefix(v, "$2a$"),
		strings.HasPrefix(v, "$2b$"),
		strings.HasPrefix(v, "$2y$"):
		return len(v) >= 50 // bcrypt is always 60 chars
	default:
		return false
	}
}

// String returns the underlying hash value (for persistence only).
func (p PasswordHash) String() string { return p.value }

// IsZero reports whether this is the zero-value hash.
func (p PasswordHash) IsZero() bool { return p.value == "" }
