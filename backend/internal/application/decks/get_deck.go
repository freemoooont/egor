package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
)

// GetDeck reads a deck for the calling owner.
type GetDeck struct {
	Decks decks.Decks
}

// Handle resolves and authorises the read.
func (uc GetDeck) Handle(ctx context.Context, in GetDeckInput) (GetDeckOutput, error) {
	d, err := uc.Decks.ByID(ctx, in.DeckID)
	if err != nil {
		return GetDeckOutput{}, err
	}
	if d.OwnerID() != in.OwnerID {
		return GetDeckOutput{}, decks.ErrForbidden
	}
	if d.IsDeleted() {
		return GetDeckOutput{}, decks.ErrDeckNotFound
	}
	return GetDeckOutput{Deck: deckViewOf(d)}, nil
}
