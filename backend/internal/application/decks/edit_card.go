package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/shared"
)

// EditCard mutates a card's term and/or definition.
type EditCard struct {
	Decks  decks.Decks
	Clock  shared.Clock
	UoW    UnitOfWork
	Events EventPublisher
}

// Handle applies the patch.
func (uc EditCard) Handle(ctx context.Context, in EditCardInput) (EditCardOutput, error) {
	var (
		newTerm *decks.Term
		newDef  *decks.Definition
	)
	if in.Term != nil {
		t, err := decks.NewTerm(*in.Term)
		if err != nil {
			return EditCardOutput{}, err
		}
		newTerm = &t
	}
	if in.Definition != nil {
		d, err := decks.NewDefinition(*in.Definition)
		if err != nil {
			return EditCardOutput{}, err
		}
		newDef = &d
	}

	var out EditCardOutput
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		d, err := uc.Decks.ByID(ctx, in.DeckID)
		if err != nil {
			return err
		}
		oldCards := d.Cards()
		var oldTerm, oldDef string
		for _, c := range oldCards {
			if c.ID() == in.CardID {
				oldTerm = c.Term().String()
				oldDef = c.Definition().String()
				break
			}
		}
		c, err := d.EditCard(in.OwnerID, in.CardID, newTerm, newDef)
		if err != nil {
			return err
		}
		if err := uc.Decks.Save(ctx, d); err != nil {
			return err
		}
		now := uc.Clock.Now(ctx)
		out = EditCardOutput{Card: cardViewOf(c)}
		return uc.Events.Publish(ctx, decks.CardEdited{
			DeckID: d.ID(), CardID: c.ID(),
			OldTerm: oldTerm, NewTerm: c.Term().String(),
			OldDefinition: oldDef, NewDefinition: c.Definition().String(),
			EditedAt: now,
		})
	})
	if err != nil {
		return EditCardOutput{}, err
	}
	return out, nil
}
