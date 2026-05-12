package decks

import (
	"context"
	"errors"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/shared"
)

// RemoveCard removes a card from a deck and recompacts ordinals.
type RemoveCard struct {
	Decks  decks.Decks
	Clock  shared.Clock
	UoW    UnitOfWork
	Events EventPublisher
}

// Handle removes the card; idempotent (unknown card id is a no-op).
func (uc RemoveCard) Handle(ctx context.Context, in RemoveCardInput) (RemoveCardOutput, error) {
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		d, err := uc.Decks.ByID(ctx, in.DeckID)
		if err != nil {
			return err
		}
		err = d.RemoveCard(in.OwnerID, in.CardID)
		if err != nil {
			if errors.Is(err, decks.ErrCardNotFound) {
				return nil
			}
			return err
		}
		if err := uc.Decks.Save(ctx, d); err != nil {
			return err
		}
		now := uc.Clock.Now(ctx)
		evs := []decks.Event{decks.CardRemoved{
			DeckID: d.ID(), CardID: in.CardID, RemovedAt: now,
		}}
		// Surface a CardsReordered event when recompaction reordered the rest.
		if d.CardCount() > 0 {
			ids := make([]string, 0, d.CardCount())
			for _, c := range d.Cards() {
				ids = append(ids, c.ID())
			}
			evs = append(evs, decks.CardsReordered{
				DeckID: d.ID(), OrderedIDs: ids, ReorderedAt: now,
			})
		}
		return uc.Events.Publish(ctx, evs...)
	})
	if err != nil {
		return RemoveCardOutput{}, err
	}
	return RemoveCardOutput{OK: true}, nil
}
