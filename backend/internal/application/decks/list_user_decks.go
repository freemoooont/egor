package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
)

// ListUserDecks returns the caller's decks (paginated).
type ListUserDecks struct {
	Decks decks.Decks
}

// DefaultListLimit is the page size when caller does not supply one.
const DefaultListLimit = 20

// Handle returns a page.
func (uc ListUserDecks) Handle(ctx context.Context, in ListUserDecksInput) (ListUserDecksOutput, error) {
	if in.OwnerID == "" {
		return ListUserDecksOutput{}, decks.ErrForbidden
	}
	limit := in.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}
	rows, next, err := uc.Decks.ByOwner(ctx, in.OwnerID, limit, in.Cursor)
	if err != nil {
		return ListUserDecksOutput{}, err
	}
	out := ListUserDecksOutput{NextCursor: next, Decks: make([]DeckSummary, 0, len(rows))}
	for _, d := range rows {
		out.Decks = append(out.Decks, DeckSummary{
			DeckID: d.ID(), Title: d.Title().String(), CardCount: d.CardCount(), CreatedAt: d.CreatedAt(),
		})
	}
	return out, nil
}
