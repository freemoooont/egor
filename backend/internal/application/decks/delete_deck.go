package decks

import (
	"context"
	"errors"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/shared"
)

// DeleteDeck soft-deletes a deck.
type DeleteDeck struct {
	Decks  decks.Decks
	Clock  shared.Clock
	UoW    UnitOfWork
	Events EventPublisher
}

// Handle marks the deck deleted; idempotent.
func (uc DeleteDeck) Handle(ctx context.Context, in DeleteDeckInput) (DeleteDeckOutput, error) {
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		d, err := uc.Decks.ByID(ctx, in.DeckID)
		if err != nil {
			if errors.Is(err, decks.ErrDeckNotFound) {
				return nil // idempotent
			}
			return err
		}
		if d.IsDeleted() {
			return nil
		}
		now := uc.Clock.Now(ctx)
		if err := d.Delete(in.OwnerID, now); err != nil {
			return err
		}
		if err := uc.Decks.Save(ctx, d); err != nil {
			return err
		}
		return uc.Events.Publish(ctx, decks.DeckDeleted{
			DeckID: d.ID(), OwnerID: d.OwnerID(), DeletedAt: now,
		})
	})
	if err != nil {
		return DeleteDeckOutput{}, err
	}
	return DeleteDeckOutput{OK: true}, nil
}
