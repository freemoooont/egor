package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestRefreshMinter_PlainAndHashAlign(t *testing.T) {
	m := NewRefreshMinter()
	plain, hash, err := m.Mint(context.Background())
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if plain == "" || hash == "" {
		t.Fatalf("empty plain/hash: %q / %q", plain, hash)
	}
	sum := sha256.Sum256([]byte(plain))
	if hex.EncodeToString(sum[:]) != hash {
		t.Fatalf("hash mismatch")
	}
}

func TestRefreshMinter_DifferentBetweenCalls(t *testing.T) {
	m := NewRefreshMinter()
	a, _, _ := m.Mint(context.Background())
	b, _, _ := m.Mint(context.Background())
	if a == b {
		t.Fatal("expected different plaintext on consecutive Mints")
	}
}

func TestHashOpaque_Stable(t *testing.T) {
	if HashOpaque("a") != HashOpaque("a") {
		t.Fatal("HashOpaque must be deterministic")
	}
	if HashOpaque("a") == HashOpaque("b") {
		t.Fatal("HashOpaque must differ for different inputs")
	}
}
