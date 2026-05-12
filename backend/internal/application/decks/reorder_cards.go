package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/shared"
)

// ReorderCards rewrites every card ordinal from the supplied permutation.
type ReorderCards struct {
	Decks  decks.Decks
	Clock  shared.Clock
	UoW    UnitOfWork
	Events EventPublisher
}

// Handle rewrites ordinals.
func (uc ReorderCards) Handle(ctx context.Context, in ReorderCardsInput) (ReorderCardsOutput, error) {
	var out ReorderCardsOutput
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		d, err := uc.Decks.ByID(ctx, in.DeckID)
		if err != nil {
			return err
		}
		if err := d.ReorderCards(in.OwnerID, in.OrderedIDs); err != nil {
			return err
		}
		if err := uc.Decks.Save(ctx, d); err != nil {
			return err
		}
		now := uc.Clock.Now(ctx)
		cards := d.Cards()
		view := make([]CardView, 0, len(cards))
		for _, c := range cards {
			view = append(view, cardViewOf(c))
		}
		out = ReorderCardsOutput{Cards: view}
		return uc.Events.Publish(ctx, decks.CardsReordered{
			DeckID: d.ID(), OrderedIDs: in.OrderedIDs, ReorderedAt: now,
		})
	})
	if err != nil {
		return ReorderCardsOutput{}, err
	}
	return out, nil
}
