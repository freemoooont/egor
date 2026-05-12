package practice

import (
	"context"
	"errors"

	"github.com/micocards/api/internal/domain/practice"
)

// GetUserDeckProgress reads the (user, deck) projection.
type GetUserDeckProgress struct {
	Progress practice.UserDeckProgresses
}

// Handle returns the projection (empty when there's nothing yet).
func (uc GetUserDeckProgress) Handle(ctx context.Context, in GetUserDeckProgressInput) (GetUserDeckProgressOutput, error) {
	if in.UserID == "" {
		return GetUserDeckProgressOutput{}, practice.ErrForbidden
	}
	p, err := uc.Progress.ByUserAndDeck(ctx, in.UserID, in.DeckID)
	if err != nil {
		// Not-found is not an error here — just an empty projection.
		var notFoundErr interface{ NotFound() bool }
		if errors.As(err, &notFoundErr) && notFoundErr.NotFound() {
			return GetUserDeckProgressOutput{DeckID: in.DeckID}, nil
		}
		return GetUserDeckProgressOutput{}, err
	}
	if p == nil {
		return GetUserDeckProgressOutput{DeckID: in.DeckID}, nil
	}
	out := GetUserDeckProgressOutput{DeckID: p.DeckID, CardProgress: make([]CardProgressView, 0, len(p.Cards))}
	for _, c := range p.Cards {
		out.CardProgress = append(out.CardProgress, CardProgressView{
			CardID: c.CardID, Rating: int16(c.Rating), LastRatedAt: c.LastRatedAt,
		})
	}
	return out, nil
}
