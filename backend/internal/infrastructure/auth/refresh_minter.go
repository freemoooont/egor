package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

// RefreshMinter implements iam.RefreshTokenMinter. Mint produces a 32-byte
// crypto/rand value, base64url-encoded as the plaintext, plus a sha256-hex
// hash for storage.
type RefreshMinter struct {
	rng readFn
}

// readFn lets tests inject a deterministic reader.
type readFn func(b []byte) (int, error)

// NewRefreshMinter returns a minter using crypto/rand.Read.
func NewRefreshMinter() *RefreshMinter {
	return &RefreshMinter{rng: rand.Read}
}

// NewRefreshMinterWithReader is for tests — fn must fill b with cryptographically
// random bytes (or a deterministic stream).
func NewRefreshMinterWithReader(fn readFn) *RefreshMinter {
	return &RefreshMinter{rng: fn}
}

// Mint returns (plaintext, hash). The plaintext is shown to the client once;
// the hash is what gets persisted.
func (m *RefreshMinter) Mint(_ context.Context) (string, string, error) {
	if m.rng == nil {
		return "", "", errors.New("refresh: nil rng")
	}
	var buf [32]byte
	if _, err := m.rng(buf[:]); err != nil {
		return "", "", err
	}
	plain := base64.RawURLEncoding.EncodeToString(buf[:])
	return plain, HashOpaque(plain), nil
}

// HashOpaque is the canonical sha256-hex digest used to store the opaque
// refresh-token value. Exported so the application/middleware layer can hash
// inbound tokens before lookup.
func HashOpaque(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// randomJTI is a small helper used by the JWT signer to mint a unique jti.
func randomJTI() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
