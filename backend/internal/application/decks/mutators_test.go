package decks_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appdecks "github.com/micocards/api/internal/application/decks"
	domdecks "github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
)

func seedDeck(t *testing.T, r *fakeDecks, ownerID string, withCards int) *domdecks.Deck {
	t.Helper()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	title, err := domdecks.NewDeckTitle("Seed")
	require.NoError(t, err)
	d, err := domdecks.NewDeck("d-seed", ownerID, title, now)
	require.NoError(t, err)
	for i := 0; i < withCards; i++ {
		term, _ := domdecks.NewTerm("t")
		def, _ := domdecks.NewDefinition("d")
		_, err := d.AddCard(ownerID, "c-"+itoa(i+1), term, def)
		require.NoError(t, err)
	}
	require.NoError(t, r.Save(context.Background(), d))
	return d
}

func TestRenameDeck_HappyPath(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 0)
	events := &fakeDeckEvents{}
	uc := appdecks.RenameDeck{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: events}
	out, err := uc.Handle(context.Background(), appdecks.RenameDeckInput{OwnerID: owner, DeckID: "d-seed", Title: "NewTitle"})
	require.NoError(t, err)
	require.Equal(t, "NewTitle", out.Title)
	require.Equal(t, []string{"decks.DeckRenamed"}, events.Names())
}

func TestRenameDeck_DeckNotFound(t *testing.T) {
	r := newFakeDecks()
	uc := appdecks.RenameDeck{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	_, err := uc.Handle(context.Background(), appdecks.RenameDeckInput{OwnerID: owner, DeckID: "missing", Title: "T"})
	require.ErrorIs(t, err, domdecks.ErrDeckNotFound)
}

func TestRenameDeck_InvalidTitle(t *testing.T) {
	r := newFakeDecks()
	uc := appdecks.RenameDeck{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	_, err := uc.Handle(context.Background(), appdecks.RenameDeckInput{OwnerID: owner, DeckID: "x", Title: ""})
	require.ErrorIs(t, err, domdecks.ErrInvalidDeckTitle)
}

func TestRenameDeck_Forbidden(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 0)
	uc := appdecks.RenameDeck{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	_, err := uc.Handle(context.Background(), appdecks.RenameDeckInput{OwnerID: "u-other", DeckID: "d-seed", Title: "T"})
	require.ErrorIs(t, err, domdecks.ErrForbidden)
}

func TestDeleteDeck_HappyPathAndIdempotent(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 1)
	events := &fakeDeckEvents{}
	uc := appdecks.DeleteDeck{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: events}
	out, err := uc.Handle(context.Background(), appdecks.DeleteDeckInput{OwnerID: owner, DeckID: "d-seed"})
	require.NoError(t, err)
	require.True(t, out.OK)
	require.Equal(t, []string{"decks.DeckDeleted"}, events.Names())

	// idempotent
	events.Reset()
	_, err = uc.Handle(context.Background(), appdecks.DeleteDeckInput{OwnerID: owner, DeckID: "d-seed"})
	require.NoError(t, err)
	require.Empty(t, events.Names())

	// unknown id is also a no-op
	_, err = uc.Handle(context.Background(), appdecks.DeleteDeckInput{OwnerID: owner, DeckID: "missing"})
	require.NoError(t, err)
}

func TestAddCard_HappyPath(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 0)
	events := &fakeDeckEvents{}
	uc := appdecks.AddCard{Decks: r, IDs: idgen.NewSequential("c"), Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: events}
	out, err := uc.Handle(context.Background(), appdecks.AddCardInput{OwnerID: owner, DeckID: "d-seed", Term: "term", Definition: "definition"})
	require.NoError(t, err)
	require.Equal(t, 1, out.Card.Ordinal)
	require.Equal(t, []string{"decks.CardAdded"}, events.Names())
}

func TestAddCard_RejectsInvalidTermOrDefinition(t *testing.T) {
	r := newFakeDecks()
	uc := appdecks.AddCard{Decks: r, IDs: idgen.NewSequential("c"), Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	_, err := uc.Handle(context.Background(), appdecks.AddCardInput{OwnerID: owner, DeckID: "x", Term: "", Definition: "d"})
	require.ErrorIs(t, err, domdecks.ErrInvalidTerm)
	_, err = uc.Handle(context.Background(), appdecks.AddCardInput{OwnerID: owner, DeckID: "x", Term: "t", Definition: ""})
	require.ErrorIs(t, err, domdecks.ErrInvalidDefinition)
}

func TestAddCard_DeckNotFound(t *testing.T) {
	r := newFakeDecks()
	uc := appdecks.AddCard{Decks: r, IDs: idgen.NewSequential("c"), Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	_, err := uc.Handle(context.Background(), appdecks.AddCardInput{OwnerID: owner, DeckID: "missing", Term: "t", Definition: "d"})
	require.ErrorIs(t, err, domdecks.ErrDeckNotFound)
}

func TestEditCard_HappyPath(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 1)
	events := &fakeDeckEvents{}
	uc := appdecks.EditCard{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: events}
	newTerm := "NEW"
	out, err := uc.Handle(context.Background(), appdecks.EditCardInput{OwnerID: owner, DeckID: "d-seed", CardID: "c-1", Term: &newTerm})
	require.NoError(t, err)
	require.Equal(t, "NEW", out.Card.Term)
	require.Equal(t, []string{"decks.CardEdited"}, events.Names())
}

func TestEditCard_RejectsInvalidTerm(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 1)
	uc := appdecks.EditCard{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	bad := ""
	_, err := uc.Handle(context.Background(), appdecks.EditCardInput{OwnerID: owner, DeckID: "d-seed", CardID: "c-1", Term: &bad})
	require.ErrorIs(t, err, domdecks.ErrInvalidTerm)
}

func TestEditCard_RejectsInvalidDefinition(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 1)
	uc := appdecks.EditCard{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	bad := ""
	_, err := uc.Handle(context.Background(), appdecks.EditCardInput{OwnerID: owner, DeckID: "d-seed", CardID: "c-1", Definition: &bad})
	require.ErrorIs(t, err, domdecks.ErrInvalidDefinition)
}

func TestEditCard_CardNotFound(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 1)
	uc := appdecks.EditCard{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	t2 := "x"
	_, err := uc.Handle(context.Background(), appdecks.EditCardInput{OwnerID: owner, DeckID: "d-seed", CardID: "missing", Term: &t2})
	require.ErrorIs(t, err, domdecks.ErrCardNotFound)
}

func TestRemoveCard_HappyPathEmitsReorderEventForRest(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 3)
	events := &fakeDeckEvents{}
	uc := appdecks.RemoveCard{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: events}
	_, err := uc.Handle(context.Background(), appdecks.RemoveCardInput{OwnerID: owner, DeckID: "d-seed", CardID: "c-1"})
	require.NoError(t, err)
	require.Equal(t, []string{"decks.CardRemoved", "decks.CardsReordered"}, events.Names())
}

func TestRemoveCard_LastCardOnlyEmitsCardRemoved(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 1)
	events := &fakeDeckEvents{}
	uc := appdecks.RemoveCard{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: events}
	_, err := uc.Handle(context.Background(), appdecks.RemoveCardInput{OwnerID: owner, DeckID: "d-seed", CardID: "c-1"})
	require.NoError(t, err)
	require.Equal(t, []string{"decks.CardRemoved"}, events.Names())
}

func TestRemoveCard_UnknownCardIsIdempotent(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 1)
	events := &fakeDeckEvents{}
	uc := appdecks.RemoveCard{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: events}
	out, err := uc.Handle(context.Background(), appdecks.RemoveCardInput{OwnerID: owner, DeckID: "d-seed", CardID: "missing"})
	require.NoError(t, err)
	require.True(t, out.OK)
	require.Empty(t, events.Names())
}

func TestReorderCards_HappyPath(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 3)
	events := &fakeDeckEvents{}
	uc := appdecks.ReorderCards{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: events}
	out, err := uc.Handle(context.Background(), appdecks.ReorderCardsInput{OwnerID: owner, DeckID: "d-seed", OrderedIDs: []string{"c-3", "c-1", "c-2"}})
	require.NoError(t, err)
	require.Equal(t, "c-3", out.Cards[0].ID)
	require.Equal(t, 1, out.Cards[0].Ordinal)
	require.Equal(t, []string{"decks.CardsReordered"}, events.Names())
}

func TestReorderCards_RejectsInvalidPermutation(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 3)
	uc := appdecks.ReorderCards{Decks: r, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeDecksUoW{}, Events: &fakeDeckEvents{}}
	_, err := uc.Handle(context.Background(), appdecks.ReorderCardsInput{OwnerID: owner, DeckID: "d-seed", OrderedIDs: []string{"c-1"}})
	require.ErrorIs(t, err, domdecks.ErrInvalidCardReorder)
}

func TestGetDeck_HappyAndForbidden(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 2)
	uc := appdecks.GetDeck{Decks: r}
	out, err := uc.Handle(context.Background(), appdecks.GetDeckInput{OwnerID: owner, DeckID: "d-seed"})
	require.NoError(t, err)
	require.Len(t, out.Deck.Cards, 2)

	_, err = uc.Handle(context.Background(), appdecks.GetDeckInput{OwnerID: "u-other", DeckID: "d-seed"})
	require.ErrorIs(t, err, domdecks.ErrForbidden)

	_, err = uc.Handle(context.Background(), appdecks.GetDeckInput{OwnerID: owner, DeckID: "missing"})
	require.ErrorIs(t, err, domdecks.ErrDeckNotFound)
}

func TestGetDeck_DeletedDeckReturnsNotFound(t *testing.T) {
	r := newFakeDecks()
	d := seedDeck(t, r, owner, 0)
	require.NoError(t, d.Delete(owner, time.Now().UTC()))
	require.NoError(t, r.Save(context.Background(), d))
	uc := appdecks.GetDeck{Decks: r}
	_, err := uc.Handle(context.Background(), appdecks.GetDeckInput{OwnerID: owner, DeckID: d.ID()})
	require.ErrorIs(t, err, domdecks.ErrDeckNotFound)
}

func TestListUserDecks_HappyPath(t *testing.T) {
	r := newFakeDecks()
	seedDeck(t, r, owner, 1)
	uc := appdecks.ListUserDecks{Decks: r}
	out, err := uc.Handle(context.Background(), appdecks.ListUserDecksInput{OwnerID: owner})
	require.NoError(t, err)
	require.Len(t, out.Decks, 1)
	require.Equal(t, 1, out.Decks[0].CardCount)
}

func TestListUserDecks_ForbiddenWhenOwnerEmpty(t *testing.T) {
	uc := appdecks.ListUserDecks{Decks: newFakeDecks()}
	_, err := uc.Handle(context.Background(), appdecks.ListUserDecksInput{})
	require.ErrorIs(t, err, domdecks.ErrForbidden)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	const digits = "0123456789"
	out := []byte{}
	for i > 0 {
		out = append([]byte{digits[i%10]}, out...)
		i /= 10
	}
	return string(out)
}
