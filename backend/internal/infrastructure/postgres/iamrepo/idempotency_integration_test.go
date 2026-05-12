//go:build integration

package iamrepo_test

import (
	"context"
	"testing"
	"time"

	domiam "github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/infrastructure/postgres/iamrepo"
	"github.com/micocards/api/internal/infrastructure/postgres/testdb"
)

func TestIdempotencyKeys_RoundTrip(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := iamrepo.NewIdempotencyKeys(pool)
	entry := domiam.IdempotencyEntry{
		Scope: "POST:/api/decks", Key: "k-1", RequestHash: "rh",
		ResponseStatus: 201, ResponseBody: []byte(`{"ok":true}`),
		ExpiresAtUnix: time.Now().Add(24 * time.Hour).Unix(),
	}
	if err := repo.Put(context.Background(), entry); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, ok, err := repo.Get(ctx, "POST:/api/decks", "k-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("Get: ok=false, want true")
	}
	if got.RequestHash != "rh" || got.ResponseStatus != 201 {
		t.Fatalf("entry mismatch: %#v", got)
	}
	if string(got.ResponseBody) != `{"ok":true}` {
		t.Fatalf("body mismatch: %s", got.ResponseBody)
	}
}

func TestIdempotencyKeys_GetMissing(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := iamrepo.NewIdempotencyKeys(pool)
	_, ok, err := repo.Get(ctx, "POST:/api/x", "missing")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for missing key")
	}
}
