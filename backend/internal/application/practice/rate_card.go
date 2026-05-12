package practice

import (
	"context"

	"github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/domain/shared"
)

// RateCard records (or updates) a rating for one card in a session.
type RateCard struct {
	Sessions practice.Sessions
	Clock    shared.Clock
	UoW      UnitOfWork
	Events   EventPublisher
}

// Handle records the rating and emits the CardRated event.
func (uc RateCard) Handle(ctx context.Context, in RateCardInput) (RateCardOutput, error) {
	if in.UserID == "" {
		return RateCardOutput{}, practice.ErrForbidden
	}
	rating := practice.CardRating(in.Rating)
	if !rating.IsValid() {
		return RateCardOutput{}, practice.ErrInvalidRating
	}

	var out RateCardOutput
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		s, err := uc.Sessions.ByID(ctx, in.SessionID)
		if err != nil {
			return err
		}
		if err := s.Authorize(in.UserID); err != nil {
			return err
		}
		now := uc.Clock.Now(ctx)
		rc, err := s.Rate(in.CardID, rating, now)
		if err != nil {
			return err
		}
		if err := uc.Sessions.Save(ctx, s); err != nil {
			return err
		}
		out = RateCardOutput{SessionID: s.ID(), CardID: rc.CardID, Rating: int16(rc.Rating), RatedAt: rc.RatedAt}
		return uc.Events.Publish(ctx, practice.CardRated{
			SessionID: s.ID(), UserID: s.UserID(), DeckID: s.DeckID(),
			CardID: rc.CardID, Rating: int16(rc.Rating), RatedAt: rc.RatedAt,
		})
	})
	if err != nil {
		return RateCardOutput{}, err
	}
	return out, nil
}
