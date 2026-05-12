//go:build integration

package iamrepo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domiam "github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/infrastructure/auth"
	pg "github.com/micocards/api/internal/infrastructure/postgres"
	"github.com/micocards/api/internal/infrastructure/postgres/iamrepo"
	"github.com/micocards/api/internal/infrastructure/postgres/testdb"
)

func newUser(t *testing.T, id, email, plain string) *domiam.User {
	t.Helper()
	em, err := domiam.NewEmailAddress(email)
	if err != nil {
		t.Fatalf("email: %v", err)
	}
	hasher := auth.NewBcryptHasher(4, 8) // bcrypt MinCost=4 in tests
	hash, err := hasher.Hash(context.Background(), plain)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	name, err := domiam.NewDisplayName("Test User")
	if err != nil {
		t.Fatalf("name: %v", err)
	}
	u, err := domiam.NewUser(id, em, hash, name, time.Now().UTC())
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	return u
}

func TestUsersRepo_SaveAndLookup(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)

	repo := iamrepo.NewUsers(pool)
	u := newUser(t, "u-1", "alice@example.com", "longenough1")
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.ByID(ctx, "u-1")
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Email().String() != "alice@example.com" {
		t.Fatalf("email mismatch: %s", got.Email())
	}
	got2, err := repo.ByEmail(ctx, u.Email())
	if err != nil {
		t.Fatalf("ByEmail: %v", err)
	}
	if got2.ID() != "u-1" {
		t.Fatalf("byEmail id mismatch: %s", got2.ID())
	}
	taken, err := repo.EmailExists(ctx, u.Email())
	if err != nil || !taken {
		t.Fatalf("EmailExists: ok=%v err=%v", taken, err)
	}
}

func TestUsersRepo_NotFound(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)

	repo := iamrepo.NewUsers(pool)
	if _, err := repo.ByID(ctx, "missing"); !errors.Is(err, domiam.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
	em, _ := domiam.NewEmailAddress("nobody@example.com")
	if _, err := repo.ByEmail(ctx, em); !errors.Is(err, domiam.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestUsersRepo_DuplicateEmail(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := iamrepo.NewUsers(pool)
	a := newUser(t, "u-a", "dup@example.com", "longenough1")
	b := newUser(t, "u-b", "dup@example.com", "longenough2")
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := repo.Save(ctx, b); !errors.Is(err, domiam.ErrEmailTaken) {
		t.Fatalf("Save b dup email: want ErrEmailTaken, got %v", err)
	}
}

func TestUsersRepo_UpdateProfileViaSave(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := iamrepo.NewUsers(pool)
	u := newUser(t, "u-up", "up@example.com", "longenough1")
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("Save: %v", err)
	}
	newName, _ := domiam.NewDisplayName("Updated Name")
	if err := u.SetDisplayName(newName); err != nil {
		t.Fatalf("SetDisplayName: %v", err)
	}
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("Save update: %v", err)
	}
	got, _ := repo.ByID(ctx, "u-up")
	if got.DisplayName().String() != "Updated Name" {
		t.Fatalf("display name not updated: %s", got.DisplayName())
	}
}

// TestUsersRepo_RoundTripWithUoW asserts the repo participates in the active tx.
func TestUsersRepo_RoundTripWithUoW(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)

	repo := iamrepo.NewUsers(pool)
	uow := pg.NewUnitOfWork(pool)
	u := newUser(t, "u-tx", "tx@example.com", "longenough1")
	err := uow.Do(ctx, func(ctx context.Context) error {
		return repo.Save(ctx, u)
	})
	if err != nil {
		t.Fatalf("UoW.Do: %v", err)
	}
	got, err := repo.ByID(ctx, "u-tx")
	if err != nil || got.ID() != "u-tx" {
		t.Fatalf("post-commit lookup: %v / %#v", err, got)
	}
}
