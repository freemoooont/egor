package decks_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appdecks "github.com/micocards/api/internal/application/decks"
	domdecks "github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
)

const owner = "u-owner"

func newCreate(t *testing.T) (appdecks.CreateDeck, *fakeDecks, *fakeDeckEvents) {
	t.Helper()
	r := newFakeDecks()
	e := &fakeDeckEvents{}
	uc := appdecks.CreateDeck{
		Decks:  r,
		IDs:    idgen.NewSequential("d"),
		Clock:  clock.NewFixed(time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)),
		UoW:    fakeDecksUoW{},
		Events: e,
	}
	return uc, r, e
}

func TestCreateDeck_HappyPathWithCards(t *testing.T) {
	uc, repo, events := newCreate(t)
	out, err := uc.Handle(context.Background(), appdecks.CreateDeckInput{
		OwnerID: owner, Title: "MyDeck",
		Cards: []appdecks.DraftCard{
			{Term: "t1", Definition: "d1"},
			{Term: "t2", Definition: "d2"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "MyDeck", out.Deck.Title)
	require.Len(t, out.Deck.Cards, 2)
	require.Equal(t, []string{"decks.DeckCreated", "decks.CardAdded", "decks.CardAdded"}, events.Names())
	require.Len(t, repo.rec, 1)
}

func TestCreateDeck_RejectsInvalidTitle(t *testing.T) {
	uc, _, _ := newCreate(t)
	_, err := uc.Handle(context.Background(), appdecks.CreateDeckInput{OwnerID: owner, Title: ""})
	require.ErrorIs(t, err, domdecks.ErrInvalidDeckTitle)
	_, err = uc.Handle(context.Background(), appdecks.CreateDeckInput{OwnerID: owner, Title: strings.Repeat("x", 121)})
	require.ErrorIs(t, err, domdecks.ErrDeckTitleTooLong)
}

func TestCreateDeck_RejectsInvalidCardContents(t *testing.T) {
	uc, _, _ := newCreate(t)
	_, err := uc.Handle(context.Background(), appdecks.CreateDeckInput{
		OwnerID: owner, Title: "T",
		Cards:   []appdecks.DraftCard{{Term: "", Definition: "d"}},
	})
	require.ErrorIs(t, err, domdecks.ErrInvalidTerm)
	_, err = uc.Handle(context.Background(), appdecks.CreateDeckInput{
		OwnerID: owner, Title: "T",
		Cards:   []appdecks.DraftCard{{Term: "t", Definition: ""}},
	})
	require.ErrorIs(t, err, domdecks.ErrInvalidDefinition)
}

func TestCreateDeck_RejectsTooManyCards(t *testing.T) {
	uc, _, _ := newCreate(t)
	cards := make([]appdecks.DraftCard, domdecks.MaxCardsPerDeck+1)
	for i := range cards {
		cards[i] = appdecks.DraftCard{Term: "t", Definition: "d"}
	}
	_, err := uc.Handle(context.Background(), appdecks.CreateDeckInput{OwnerID: owner, Title: "T", Cards: cards})
	require.ErrorIs(t, err, domdecks.ErrDeckCardLimitExceeded)
}

func TestCreateDeck_ForbiddenWhenOwnerEmpty(t *testing.T) {
	uc, _, _ := newCreate(t)
	_, err := uc.Handle(context.Background(), appdecks.CreateDeckInput{OwnerID: "", Title: "T"})
	require.ErrorIs(t, err, domdecks.ErrForbidden)
}
