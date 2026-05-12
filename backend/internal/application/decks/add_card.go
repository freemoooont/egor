package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/shared"
)

// AddCard appends a card to a deck.
type AddCard struct {
	Decks  decks.Decks
	IDs    shared.IDGenerator
	Clock  shared.Clock
	UoW    UnitOfWork
	Events EventPublisher
}

// Handle appends the card.
func (uc AddCard) Handle(ctx context.Context, in AddCardInput) (AddCardOutput, error) {
	term, err := decks.NewTerm(in.Term)
	if err != nil {
		return AddCardOutput{}, err
	}
	def, err := decks.NewDefinition(in.Definition)
	if err != nil {
		return AddCardOutput{}, err
	}
	var out AddCardOutput
	err = uc.UoW.Do(ctx, func(ctx context.Context) error {
		d, err := uc.Decks.ByID(ctx, in.DeckID)
		if err != nil {
			return err
		}
		cardID := uc.IDs.NewID(ctx)
		c, err := d.AddCard(in.OwnerID, cardID, term, def)
		if err != nil {
			return err
		}
		if err := uc.Decks.Save(ctx, d); err != nil {
			return err
		}
		now := uc.Clock.Now(ctx)
		out = AddCardOutput{Card: cardViewOf(c)}
		return uc.Events.Publish(ctx, decks.CardAdded{
			DeckID: d.ID(), CardID: c.ID(), Term: c.Term().String(),
			Definition: c.Definition().String(), Ordinal: c.Ordinal(), AddedAt: now,
		})
	})
	if err != nil {
		return AddCardOutput{}, err
	}
	return out, nil
}
