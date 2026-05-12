package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/shared"
)

// CreateDeck creates a deck with optional bundled cards.
type CreateDeck struct {
	Decks  decks.Decks
	IDs    shared.IDGenerator
	Clock  shared.Clock
	UoW    UnitOfWork
	Events EventPublisher
}

// Handle validates the input, builds the Deck, and persists it.
func (uc CreateDeck) Handle(ctx context.Context, in CreateDeckInput) (CreateDeckOutput, error) {
	if in.OwnerID == "" {
		return CreateDeckOutput{}, decks.ErrForbidden
	}
	title, err := decks.NewDeckTitle(in.Title)
	if err != nil {
		return CreateDeckOutput{}, err
	}
	if len(in.Cards) > decks.MaxCardsPerDeck {
		return CreateDeckOutput{}, decks.ErrDeckCardLimitExceeded
	}

	var out CreateDeckOutput
	err = uc.UoW.Do(ctx, func(ctx context.Context) error {
		now := uc.Clock.Now(ctx)
		deckID := uc.IDs.NewID(ctx)
		deck, err := decks.NewDeck(deckID, in.OwnerID, title, now)
		if err != nil {
			return err
		}
		evs := []decks.Event{decks.DeckCreated{
			DeckID: deck.ID(), OwnerID: deck.OwnerID(), Title: deck.Title().String(),
			CardCount: len(in.Cards), CreatedAt: now,
		}}
		for _, dc := range in.Cards {
			term, err := decks.NewTerm(dc.Term)
			if err != nil {
				return err
			}
			def, err := decks.NewDefinition(dc.Definition)
			if err != nil {
				return err
			}
			cardID := uc.IDs.NewID(ctx)
			card, err := deck.AddCard(in.OwnerID, cardID, term, def)
			if err != nil {
				return err
			}
			evs = append(evs, decks.CardAdded{
				DeckID: deck.ID(), CardID: card.ID(), Term: term.String(),
				Definition: def.String(), Ordinal: card.Ordinal(), AddedAt: now,
			})
		}
		if err := uc.Decks.Save(ctx, deck); err != nil {
			return err
		}
		out = CreateDeckOutput{Deck: deckViewOf(deck)}
		return uc.Events.Publish(ctx, evs...)
	})
	if err != nil {
		return CreateDeckOutput{}, err
	}
	return out, nil
}
