package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/shared"
)

// RenameDeck mutates a deck's title.
type RenameDeck struct {
	Decks  decks.Decks
	Clock  shared.Clock
	UoW    UnitOfWork
	Events EventPublisher
}

// Handle validates and applies the rename.
func (uc RenameDeck) Handle(ctx context.Context, in RenameDeckInput) (RenameDeckOutput, error) {
	title, err := decks.NewDeckTitle(in.Title)
	if err != nil {
		return RenameDeckOutput{}, err
	}
	var out RenameDeckOutput
	err = uc.UoW.Do(ctx, func(ctx context.Context) error {
		d, err := uc.Decks.ByID(ctx, in.DeckID)
		if err != nil {
			return err
		}
		oldTitle := d.Title().String()
		if err := d.Rename(in.OwnerID, title); err != nil {
			return err
		}
		if err := uc.Decks.Save(ctx, d); err != nil {
			return err
		}
		now := uc.Clock.Now(ctx)
		out = RenameDeckOutput{DeckID: d.ID(), Title: d.Title().String(), RenamedAt: now}
		return uc.Events.Publish(ctx, decks.DeckRenamed{
			DeckID: d.ID(), OwnerID: d.OwnerID(),
			OldTitle: oldTitle, NewTitle: d.Title().String(), RenamedAt: now,
		})
	})
	if err != nil {
		return RenameDeckOutput{}, err
	}
	return out, nil
}
