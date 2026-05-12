//go:build integration

package decksrepo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/micocards/api/internal/domain/decks"
	pg "github.com/micocards/api/internal/infrastructure/postgres"
	"github.com/micocards/api/internal/infrastructure/postgres/decksrepo"
	"github.com/micocards/api/internal/infrastructure/postgres/testdb"
)

func newDeck(t *testing.T, id, owner, title string) *decks.Deck {
	t.Helper()
	tt, err := decks.NewDeckTitle(title)
	if err != nil {
		t.Fatalf("title: %v", err)
	}
	d, err := decks.NewDeck(id, owner, tt, time.Now().UTC())
	if err != nil {
		t.Fatalf("NewDeck: %v", err)
	}
	return d
}

func TestDecks_RoundTripWithCards(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := decksrepo.NewDecks(pool)
	uow := pg.NewUnitOfWork(pool)

	d := newDeck(t, "d-1", "u-1", "Capitals")
	term, _ := decks.NewTerm("France")
	def, _ := decks.NewDefinition("Paris")
	if _, err := d.AddCard("u-1", "c-1", term, def); err != nil {
		t.Fatalf("AddCard: %v", err)
	}

	if err := uow.Do(ctx, func(ctx context.Context) error {
		return repo.Save(ctx, d)
	}); err != nil {
		t.Fatalf("UoW Save: %v", err)
	}
	got, err := repo.ByID(ctx, "d-1")
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Title().String() != "Capitals" {
		t.Fatalf("title: %s", got.Title())
	}
	if got.CardCount() != 1 {
		t.Fatalf("card count: %d", got.CardCount())
	}
	cards := got.Cards()
	if cards[0].Term().String() != "France" || cards[0].Definition().String() != "Paris" {
		t.Fatalf("card content: %#v", cards[0])
	}
	if cards[0].Ordinal() != 1 {
		t.Fatalf("ordinal: %d", cards[0].Ordinal())
	}
}

func TestDecks_ByOwner(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := decksrepo.NewDecks(pool)
	uow := pg.NewUnitOfWork(pool)
	for _, id := range []string{"d-a", "d-b", "d-c"} {
		d := newDeck(t, id, "u-owner", "Title-"+id)
		if err := uow.Do(ctx, func(ctx context.Context) error { return repo.Save(ctx, d) }); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	got, _, err := repo.ByOwner(ctx, "u-owner", 10, "")
	if err != nil {
		t.Fatalf("ByOwner: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 decks, got %d", len(got))
	}
}

func TestDecks_SoftDeleteHidesFromByOwner(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := decksrepo.NewDecks(pool)
	uow := pg.NewUnitOfWork(pool)
	d := newDeck(t, "d-del", "u-owner", "Doomed")
	if err := uow.Do(ctx, func(ctx context.Context) error { return repo.Save(ctx, d) }); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := d.Delete("u-owner", time.Now().UTC()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := uow.Do(ctx, func(ctx context.Context) error { return repo.Save(ctx, d) }); err != nil {
		t.Fatalf("Save delete: %v", err)
	}
	got, _, err := repo.ByOwner(ctx, "u-owner", 10, "")
	if err != nil {
		t.Fatalf("ByOwner: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want 0 (deleted hidden), got %d", len(got))
	}
	// ByID still returns it (soft delete).
	d2, err := repo.ByID(ctx, "d-del")
	if err != nil {
		t.Fatalf("ByID after soft-delete: %v", err)
	}
	if !d2.IsDeleted() {
		t.Fatal("expected IsDeleted=true after persisted soft delete")
	}
}

func TestDecks_NotFound(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := decksrepo.NewDecks(pool)
	if _, err := repo.ByID(ctx, "missing"); !errors.Is(err, decks.ErrDeckNotFound) {
		t.Fatalf("want ErrDeckNotFound, got %v", err)
	}
}

func TestDecks_OwnerAndCardsSnapshot(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := decksrepo.NewDecks(pool)
	uow := pg.NewUnitOfWork(pool)
	d := newDeck(t, "d-snap", "u-1", "Snap")
	for i, pair := range [][2]string{{"a", "1"}, {"b", "2"}, {"c", "3"}} {
		term, _ := decks.NewTerm(pair[0])
		def, _ := decks.NewDefinition(pair[1])
		if _, err := d.AddCard("u-1", "c-"+string(rune('1'+i)), term, def); err != nil {
			t.Fatalf("AddCard: %v", err)
		}
	}
	if err := uow.Do(ctx, func(ctx context.Context) error { return repo.Save(ctx, d) }); err != nil {
		t.Fatalf("Save: %v", err)
	}
	owner, ids, err := repo.OwnerAndCards(ctx, "d-snap")
	if err != nil {
		t.Fatalf("OwnerAndCards: %v", err)
	}
	if owner != "u-1" {
		t.Fatalf("owner: %s", owner)
	}
	if len(ids) != 3 {
		t.Fatalf("want 3 card ids, got %d", len(ids))
	}
}
