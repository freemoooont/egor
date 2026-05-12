//go:build integration

package iamrepo_test

import (
	"errors"
	"testing"
	"time"

	domiam "github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/infrastructure/postgres/iamrepo"
	"github.com/micocards/api/internal/infrastructure/postgres/testdb"
)

func newToken(id, family, user, hash string, issued time.Time, ttl time.Duration) domiam.RefreshToken {
	return domiam.RefreshToken{
		ID: id, FamilyID: family, UserID: user, OpaqueHash: hash,
		IssuedAt: issued, ExpiresAt: issued.Add(ttl),
	}
}

func TestRefreshTokens_SaveAndLookup(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := iamrepo.NewRefreshTokens(pool, nil)
	tok := newToken("t-1", "f-1", "u-1", "hash1", time.Now().UTC(), time.Hour)
	if err := repo.Save(ctx, tok); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.ByOpaqueHash(ctx, "hash1")
	if err != nil {
		t.Fatalf("ByOpaqueHash: %v", err)
	}
	if got.ID != "t-1" {
		t.Fatalf("id mismatch: %s", got.ID)
	}
}

func TestRefreshTokens_FamilyByID(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := iamrepo.NewRefreshTokens(pool, nil)
	now := time.Now().UTC()
	for i, h := range []string{"h1", "h2", "h3"} {
		tok := newToken(
			"t-"+string(rune('1'+i)),
			"fam", "u-1", h,
			now.Add(time.Duration(i)*time.Second),
			time.Hour,
		)
		if err := repo.Save(ctx, tok); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	fam, err := repo.FamilyByID(ctx, "fam")
	if err != nil {
		t.Fatalf("FamilyByID: %v", err)
	}
	if len(fam.Tokens) != 3 {
		t.Fatalf("want 3 tokens, got %d", len(fam.Tokens))
	}
	for i := 1; i < 3; i++ {
		if !fam.Tokens[i].IssuedAt.After(fam.Tokens[i-1].IssuedAt) {
			t.Fatal("expected issued_at strictly ascending")
		}
	}
}

func TestRefreshTokens_RevokeFamily(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := iamrepo.NewRefreshTokens(pool, nil)
	now := time.Now().UTC()
	if err := repo.Save(ctx, newToken("t1", "f", "u", "h1", now, time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, newToken("t2", "f", "u", "h2", now.Add(time.Second), time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repo.RevokeFamily(ctx, "f", nil, domiam.RevokeReasonLogout); err != nil {
		t.Fatalf("RevokeFamily: %v", err)
	}
	fam, _ := repo.FamilyByID(ctx, "f")
	for _, tok := range fam.Tokens {
		if tok.RevokedAt == nil {
			t.Fatalf("token %s not revoked", tok.ID)
		}
	}
}

func TestRefreshTokens_RevokeOne_MissingErrors(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := iamrepo.NewRefreshTokens(pool, nil)
	if err := repo.RevokeOne(ctx, "missing", nil, domiam.RevokeReasonRotation); !errors.Is(err, domiam.ErrRefreshTokenInvalid) {
		t.Fatalf("missing token revoke: want ErrRefreshTokenInvalid, got %v", err)
	}
}
