package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domiam "github.com/micocards/api/internal/domain/iam"
)

func newSigner(t *testing.T, opts ...SignerOption) *AccessTokenSigner {
	t.Helper()
	s, err := NewAccessTokenSigner([]byte("0123456789abcdef0123456789abcdef"), 15*time.Minute, opts...)
	if err != nil {
		t.Fatalf("NewAccessTokenSigner: %v", err)
	}
	return s
}

func TestAccessTokenSigner_RoundTrip(t *testing.T) {
	s := newSigner(t)
	tok, exp, err := s.SignAccessToken(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	if tok == "" || exp == 0 {
		t.Fatalf("empty token/exp: %q / %d", tok, exp)
	}
	sub, err := s.VerifyAccessToken(context.Background(), tok)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}
	if sub != "u-1" {
		t.Fatalf("want u-1, got %q", sub)
	}
}

func TestAccessTokenSigner_RejectsExpired(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	s := newSigner(t, WithSignerNow(func() time.Time { return now }))
	tok, _, _ := s.SignAccessToken(context.Background(), "u-1")
	// Move clock past expiry.
	*s = *newSigner(t, WithSignerNow(func() time.Time { return now.Add(time.Hour) }))
	if _, err := s.VerifyAccessToken(context.Background(), tok); !errors.Is(err, domiam.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}

func TestAccessTokenSigner_RejectsBadSignature(t *testing.T) {
	a := newSigner(t)
	tok, _, _ := a.SignAccessToken(context.Background(), "u-1")
	b, _ := NewAccessTokenSigner([]byte("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"), 15*time.Minute)
	if _, err := b.VerifyAccessToken(context.Background(), tok); !errors.Is(err, domiam.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}

func TestAccessTokenSigner_RequiresStrongSecret(t *testing.T) {
	if _, err := NewAccessTokenSigner([]byte("short"), 15*time.Minute); err == nil {
		t.Fatal("expected error for short secret")
	}
}
