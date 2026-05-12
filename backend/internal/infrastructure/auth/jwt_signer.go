package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	domiam "github.com/micocards/api/internal/domain/iam"
)

// AccessTokenSigner implements iam.AccessTokenSigner using HS256.
type AccessTokenSigner struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
	idgen  func() string
}

// SignerOption configures an AccessTokenSigner.
type SignerOption func(*AccessTokenSigner)

// WithSignerNow overrides the wall clock (used by tests).
func WithSignerNow(fn func() time.Time) SignerOption {
	return func(s *AccessTokenSigner) { s.now = fn }
}

// WithSignerJTI overrides the jti generator (used by tests).
func WithSignerJTI(fn func() string) SignerOption {
	return func(s *AccessTokenSigner) { s.idgen = fn }
}

// NewAccessTokenSigner builds a signer.
//   - secret must be at least 32 bytes (HMAC-SHA256 key spec).
//   - ttl is the access-token lifetime; ADR 0003 fixes it at 15 minutes.
func NewAccessTokenSigner(secret []byte, ttl time.Duration, opts ...SignerOption) (*AccessTokenSigner, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt: secret must be >=32 bytes")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	s := &AccessTokenSigner{
		secret: secret,
		ttl:    ttl,
		now:    func() time.Time { return time.Now().UTC() },
		idgen:  randomJTI,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// SignAccessToken returns the encoded JWT and its absolute expiry (unix sec).
func (s *AccessTokenSigner) SignAccessToken(_ context.Context, userID string) (string, int64, error) {
	if userID == "" {
		return "", 0, errors.New("jwt: empty user id")
	}
	now := s.now()
	exp := now.Add(s.ttl)
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": now.Unix(),
		"exp": exp.Unix(),
		"jti": s.idgen(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", 0, fmt.Errorf("jwt sign: %w", err)
	}
	return signed, exp.Unix(), nil
}

// VerifyAccessToken parses the token and returns the userID. Returns
// ErrUnauthorized on any failure (signature, expiry, malformed).
func (s *AccessTokenSigner) VerifyAccessToken(_ context.Context, raw string) (string, error) {
	if raw == "" {
		return "", domiam.ErrUnauthorized
	}
	parsed, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithTimeFunc(s.now), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !parsed.Valid {
		return "", domiam.ErrUnauthorized
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", domiam.ErrUnauthorized
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", domiam.ErrUnauthorized
	}
	return sub, nil
}
