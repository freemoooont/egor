package auth

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	domiam "github.com/micocards/api/internal/domain/iam"
)

func TestBcryptHasher_RoundTrip(t *testing.T) {
	h := NewBcryptHasher(bcrypt.MinCost, 8)
	hash, err := h.Hash(context.Background(), "supersecure1")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if hash.IsZero() {
		t.Fatal("expected non-zero hash")
	}
	if err := h.Compare(context.Background(), hash, "supersecure1"); err != nil {
		t.Fatalf("Compare matching: %v", err)
	}
	if err := h.Compare(context.Background(), hash, "wrongpassword"); !errors.Is(err, domiam.ErrInvalidCredentials) {
		t.Fatalf("Compare mismatch: want ErrInvalidCredentials, got %v", err)
	}
}

func TestBcryptHasher_StrengthRejectsShort(t *testing.T) {
	h := NewBcryptHasher(bcrypt.MinCost, 8)
	if err := h.Strength("short"); !errors.Is(err, domiam.ErrPasswordTooWeak) {
		t.Fatalf("Strength: want ErrPasswordTooWeak, got %v", err)
	}
	if err := h.Strength("longenough"); err != nil {
		t.Fatalf("Strength: got %v, want nil", err)
	}
}

func TestBcryptHasher_HashRejectsWeak(t *testing.T) {
	h := NewBcryptHasher(bcrypt.MinCost, 8)
	if _, err := h.Hash(context.Background(), "short"); !errors.Is(err, domiam.ErrPasswordTooWeak) {
		t.Fatalf("Hash weak: want ErrPasswordTooWeak, got %v", err)
	}
}
