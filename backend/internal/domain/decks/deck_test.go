package decks_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/domain/decks"
)

const owner = "u-owner"
const stranger = "u-stranger"

func mustTitle(t *testing.T, v string) decks.DeckTitle {
	t.Helper()
	d, err := decks.NewDeckTitle(v)
	require.NoError(t, err)
	return d
}

func mustTerm(t *testing.T, v string) decks.Term {
	t.Helper()
	x, err := decks.NewTerm(v)
	require.NoError(t, err)
	return x
}

func mustDef(t *testing.T, v string) decks.Definition {
	t.Helper()
	x, err := decks.NewDefinition(v)
	require.NoError(t, err)
	return x
}

func newDeck(t *testing.T) *decks.Deck {
	t.Helper()
	d, err := decks.NewDeck("d-1", owner, mustTitle(t, "MyDeck"), time.Now().UTC())
	require.NoError(t, err)
	return d
}

// Invariant 1.
func TestDeck_TitleLengthBetween1And120(t *testing.T) {
	_, err := decks.NewDeckTitle("")
	require.ErrorIs(t, err, decks.ErrInvalidDeckTitle)
	_, err = decks.NewDeckTitle("   ")
	require.ErrorIs(t, err, decks.ErrInvalidDeckTitle)
	_, err = decks.NewDeckTitle(strings.Repeat("x", 121))
	require.ErrorIs(t, err, decks.ErrDeckTitleTooLong)
	_, err = decks.NewDeckTitle(strings.Repeat("x", 120))
	require.NoError(t, err)
}

// Invariant 2 (struct field is unexported; OwnerID is read-only).
func TestDeck_OwnerIDIsImmutable(t *testing.T) {
	d := newDeck(t)
	require.Equal(t, owner, d.OwnerID())
	// no SetOwner method exists — compile-time assertion is implicit.
}

// Invariant 3.
func TestDeck_HasAtMost500Cards(t *testing.T) {
	d := newDeck(t)
	for i := 0; i < decks.MaxCardsPerDeck; i++ {
		_, err := d.AddCard(owner, "c-"+istr(i), mustTerm(t, "t"), mustDef(t, "d"))
		require.NoError(t, err)
	}
	_, err := d.AddCard(owner, "c-overflow", mustTerm(t, "t"), mustDef(t, "d"))
	require.ErrorIs(t, err, decks.ErrDeckCardLimitExceeded)
}

// Invariant 4 + 5 (dense, unique).
func TestDeck_CardOrdinalsAreDense1ToN_AndUnique(t *testing.T) {
	d := newDeck(t)
	for i := 0; i < 5; i++ {
		_, err := d.AddCard(owner, "c-"+istr(i), mustTerm(t, "t"), mustDef(t, "d"))
		require.NoError(t, err)
	}
	cards := d.Cards()
	require.Len(t, cards, 5)
	seen := make(map[int]bool)
	for i, c := range cards {
		require.Equal(t, i+1, c.Ordinal())
		require.False(t, seen[c.Ordinal()])
		seen[c.Ordinal()] = true
	}
}

// Invariant 6.
func TestCard_TermLengthBetween1And512(t *testing.T) {
	_, err := decks.NewTerm("")
	require.ErrorIs(t, err, decks.ErrInvalidTerm)
	_, err = decks.NewTerm(strings.Repeat("x", 513))
	require.ErrorIs(t, err, decks.ErrInvalidTerm)
	_, err = decks.NewTerm(strings.Repeat("x", 512))
	require.NoError(t, err)
}

// Invariant 7.
func TestCard_DefinitionLengthBetween1And2048(t *testing.T) {
	_, err := decks.NewDefinition("")
	require.ErrorIs(t, err, decks.ErrInvalidDefinition)
	_, err = decks.NewDefinition(strings.Repeat("x", 2049))
	require.ErrorIs(t, err, decks.ErrInvalidDefinition)
	_, err = decks.NewDefinition(strings.Repeat("x", 2048))
	require.NoError(t, err)
}

// Invariant 8.
func TestDeck_AddCardAppendsAtNextOrdinal(t *testing.T) {
	d := newDeck(t)
	c1, err := d.AddCard(owner, "c-1", mustTerm(t, "t1"), mustDef(t, "d1"))
	require.NoError(t, err)
	require.Equal(t, 1, c1.Ordinal())
	c2, err := d.AddCard(owner, "c-2", mustTerm(t, "t2"), mustDef(t, "d2"))
	require.NoError(t, err)
	require.Equal(t, 2, c2.Ordinal())
}

// Invariant 9.
func TestDeck_RemoveCardRecompactsOrdinals(t *testing.T) {
	d := newDeck(t)
	for i := 0; i < 4; i++ {
		_, err := d.AddCard(owner, "c-"+istr(i), mustTerm(t, "t"), mustDef(t, "d"))
		require.NoError(t, err)
	}
	require.NoError(t, d.RemoveCard(owner, "c-1"))
	cards := d.Cards()
	require.Len(t, cards, 3)
	for i, c := range cards {
		require.Equal(t, i+1, c.Ordinal(), "card %s should have ordinal %d", c.ID(), i+1)
	}
}

// Invariant 10.
func TestDeck_ReorderCardsRequiresExactPermutationOfExistingIDs(t *testing.T) {
	d := newDeck(t)
	for i := 0; i < 3; i++ {
		_, err := d.AddCard(owner, "c-"+istr(i), mustTerm(t, "t"), mustDef(t, "d"))
		require.NoError(t, err)
	}
	// Wrong length.
	require.ErrorIs(t, d.ReorderCards(owner, []string{"c-0", "c-1"}), decks.ErrInvalidCardReorder)
	// Duplicate.
	require.ErrorIs(t, d.ReorderCards(owner, []string{"c-0", "c-0", "c-1"}), decks.ErrInvalidCardReorder)
	// Unknown id.
	require.ErrorIs(t, d.ReorderCards(owner, []string{"c-0", "c-1", "c-XYZ"}), decks.ErrInvalidCardReorder)
	// Valid permutation.
	require.NoError(t, d.ReorderCards(owner, []string{"c-2", "c-0", "c-1"}))
	cards := d.Cards()
	require.Equal(t, "c-2", cards[0].ID())
	require.Equal(t, 1, cards[0].Ordinal())
	require.Equal(t, "c-0", cards[1].ID())
	require.Equal(t, 2, cards[1].Ordinal())
}

// Invariant 11.
func TestDeck_RenameDoesNotMutateCards(t *testing.T) {
	d := newDeck(t)
	_, err := d.AddCard(owner, "c-1", mustTerm(t, "t"), mustDef(t, "d"))
	require.NoError(t, err)
	before := d.Cards()
	require.NoError(t, d.Rename(owner, mustTitle(t, "Updated")))
	require.Equal(t, "Updated", d.Title().String())
	require.Equal(t, before, d.Cards())
}

// Invariant 12.
func TestDeck_DeleteIsTerminal(t *testing.T) {
	d := newDeck(t)
	require.NoError(t, d.Delete(owner, time.Now().UTC()))
	require.True(t, d.IsDeleted())

	require.ErrorIs(t, d.Rename(owner, mustTitle(t, "Whatever")), decks.ErrDeckDeleted)
	_, err := d.AddCard(owner, "c-1", mustTerm(t, "t"), mustDef(t, "d"))
	require.ErrorIs(t, err, decks.ErrDeckDeleted)

	// Idempotent.
	require.NoError(t, d.Delete(owner, time.Now().UTC()))
}

// Invariant 13.
func TestDeck_OnlyOwnerCanMutate(t *testing.T) {
	d := newDeck(t)
	require.ErrorIs(t, d.Rename(stranger, mustTitle(t, "x")), decks.ErrForbidden)
	_, err := d.AddCard(stranger, "c-1", mustTerm(t, "t"), mustDef(t, "d"))
	require.ErrorIs(t, err, decks.ErrForbidden)
	_, err = d.EditCard(stranger, "c-1", nil, nil)
	require.ErrorIs(t, err, decks.ErrForbidden)
	require.ErrorIs(t, d.RemoveCard(stranger, "c-1"), decks.ErrForbidden)
	require.ErrorIs(t, d.ReorderCards(stranger, nil), decks.ErrForbidden)
	require.ErrorIs(t, d.Delete(stranger, time.Now().UTC()), decks.ErrForbidden)
}

func TestDeck_EditCardChangesTermAndDefinition(t *testing.T) {
	d := newDeck(t)
	_, err := d.AddCard(owner, "c-1", mustTerm(t, "old-t"), mustDef(t, "old-d"))
	require.NoError(t, err)

	newTerm := mustTerm(t, "new-t")
	newDef := mustDef(t, "new-d")
	got, err := d.EditCard(owner, "c-1", &newTerm, &newDef)
	require.NoError(t, err)
	require.Equal(t, "new-t", got.Term().String())
	require.Equal(t, "new-d", got.Definition().String())

	// Card not found.
	_, err = d.EditCard(owner, "c-missing", &newTerm, nil)
	require.ErrorIs(t, err, decks.ErrCardNotFound)
}

func TestDeck_RemoveCardNotFound(t *testing.T) {
	d := newDeck(t)
	require.ErrorIs(t, d.RemoveCard(owner, "c-missing"), decks.ErrCardNotFound)
}

func TestDeck_NewDeckRejectsZeroValues(t *testing.T) {
	now := time.Now().UTC()
	_, err := decks.NewDeck("", owner, mustTitle(t, "T"), now)
	require.ErrorIs(t, err, decks.ErrDeckNotFound)
	_, err = decks.NewDeck("d", "", mustTitle(t, "T"), now)
	require.ErrorIs(t, err, decks.ErrDeckNotFound)
	_, err = decks.NewDeck("d", owner, decks.DeckTitle{}, now)
	require.ErrorIs(t, err, decks.ErrInvalidDeckTitle)
}

func TestDeck_HydrateRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	d := newDeck(t)
	_, err := d.AddCard(owner, "c-1", mustTerm(t, "t"), mustDef(t, "d"))
	require.NoError(t, err)
	hydrated := decks.HydrateDeck(d.ID(), d.OwnerID(), d.Title(), d.Cards(), d.CreatedAt(), nil)
	require.Equal(t, d.ID(), hydrated.ID())
	require.Equal(t, d.CardCount(), hydrated.CardCount())
	require.False(t, hydrated.IsDeleted())

	deletedAt := now.Add(time.Hour)
	hydrated2 := decks.HydrateDeck(d.ID(), d.OwnerID(), d.Title(), d.Cards(), d.CreatedAt(), &deletedAt)
	require.True(t, hydrated2.IsDeleted())
}

func TestEvents_NamesArePackagePrefixed(t *testing.T) {
	cases := []decks.Event{
		decks.DeckCreated{DeckID: "d"},
		decks.DeckRenamed{DeckID: "d"},
		decks.DeckDeleted{DeckID: "d"},
		decks.CardAdded{DeckID: "d", CardID: "c"},
		decks.CardEdited{DeckID: "d", CardID: "c"},
		decks.CardRemoved{DeckID: "d", CardID: "c"},
		decks.CardsReordered{DeckID: "d"},
		decks.DeckGenerationRequested{RequestID: "r"},
		decks.DeckGenerationCompleted{RequestID: "r"},
	}
	want := []string{
		"decks.DeckCreated",
		"decks.DeckRenamed",
		"decks.DeckDeleted",
		"decks.CardAdded",
		"decks.CardEdited",
		"decks.CardRemoved",
		"decks.CardsReordered",
		"decks.DeckGenerationRequested",
		"decks.DeckGenerationCompleted",
	}
	for i, ev := range cases {
		require.Equal(t, want[i], ev.Name())
	}
}

func istr(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	out := []byte{}
	for i > 0 {
		out = append([]byte{digits[i%10]}, out...)
		i /= 10
	}
	return string(out)
}
