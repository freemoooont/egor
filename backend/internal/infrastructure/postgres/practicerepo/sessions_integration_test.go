//go:build integration

package practicerepo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/micocards/api/internal/domain/practice"
	pg "github.com/micocards/api/internal/infrastructure/postgres"
	"github.com/micocards/api/internal/infrastructure/postgres/practicerepo"
	"github.com/micocards/api/internal/infrastructure/postgres/testdb"
)

func newSession(t *testing.T, id, user, deck string, cardIDs []string) *practice.Session {
	t.Helper()
	s, err := practice.NewSession(id, user, deck, practice.ModeTracked, cardIDs, time.Now().UTC())
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	return s
}

func TestSessions_RoundTrip(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := practicerepo.NewSessions(pool)
	uow := pg.NewUnitOfWork(pool)

	s := newSession(t, "s-1", "u-1", "d-1", []string{"c-1", "c-2", "c-3"})
	if _, err := s.Rate("c-1", practice.RatingKnowKnow, time.Now().UTC()); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if _, err := s.Rate("c-2", practice.RatingDontKnow, time.Now().UTC()); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if err := uow.Do(ctx, func(ctx context.Context) error { return repo.Save(ctx, s) }); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.ByID(ctx, "s-1")
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	sum := got.Summary()
	if sum.CountKnowKnow != 1 || sum.CountDontKnow != 1 || sum.CountStillLearning != 0 {
		t.Fatalf("summary: %#v", sum)
	}
	if got.Mode() != practice.ModeTracked {
		t.Fatalf("mode: %s", got.Mode())
	}
	if got.Status() != practice.StatusInProgress {
		t.Fatalf("status: %s", got.Status())
	}
}

func TestSessions_FinishPersists(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := practicerepo.NewSessions(pool)
	uow := pg.NewUnitOfWork(pool)
	s := newSession(t, "s-fin", "u-1", "d-1", []string{"c-1"})
	_, _ = s.Rate("c-1", practice.RatingKnowKnow, time.Now().UTC())
	if _, err := s.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if err := uow.Do(ctx, func(ctx context.Context) error { return repo.Save(ctx, s) }); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.LatestCompletedFor(ctx, "u-1", "d-1")
	if err != nil {
		t.Fatalf("LatestCompletedFor: %v", err)
	}
	if got.Status() != practice.StatusCompleted {
		t.Fatalf("status: %s", got.Status())
	}
}

func TestSessions_NotFound(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := practicerepo.NewSessions(pool)
	if _, err := repo.ByID(ctx, "missing"); !errors.Is(err, practice.ErrSessionNotFound) {
		t.Fatalf("want ErrSessionNotFound, got %v", err)
	}
	if _, err := repo.LatestCompletedFor(ctx, "u-x", "d-x"); !errors.Is(err, practice.ErrSessionNotFound) {
		t.Fatalf("want ErrSessionNotFound, got %v", err)
	}
}

func TestUserDeckProgress_RoundTrip(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	repo := practicerepo.NewUserDeckProgresses(pool)
	at := time.Now().UTC()
	p := &practice.UserDeckProgress{
		UserID: "u-1", DeckID: "d-1",
		Cards: []practice.CardProgress{
			{CardID: "c-1", Rating: practice.RatingKnowKnow, LastRatedAt: at},
			{CardID: "c-2", Rating: practice.RatingDontKnow, LastRatedAt: at},
			{CardID: "c-3", Rating: practice.RatingStillLearning, LastRatedAt: at},
		},
		UpdatedAt: at,
	}
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.ByUserAndDeck(ctx, "u-1", "d-1")
	if err != nil {
		t.Fatalf("ByUserAndDeck: %v", err)
	}
	if len(got.Cards) != 3 {
		t.Fatalf("want 3 cards, got %d", len(got.Cards))
	}

	// DeleteByDeck wipes the row.
	if err := repo.DeleteByDeck(ctx, "d-1"); err != nil {
		t.Fatalf("DeleteByDeck: %v", err)
	}
	got2, err := repo.ByUserAndDeck(ctx, "u-1", "d-1")
	if err != nil {
		t.Fatalf("ByUserAndDeck after delete: %v", err)
	}
	if !got2.IsZero() {
		t.Fatalf("want IsZero after delete, got %d cards", len(got2.Cards))
	}
}
