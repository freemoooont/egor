package practice

import (
	"context"

	"github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/domain/shared"
)

// FinishSession closes a session.
type FinishSession struct {
	Sessions practice.Sessions
	Clock    shared.Clock
	UoW      UnitOfWork
	Events   EventPublisher
}

// Handle finishes the session.
func (uc FinishSession) Handle(ctx context.Context, in FinishSessionInput) (FinishSessionOutput, error) {
	if in.UserID == "" {
		return FinishSessionOutput{}, practice.ErrForbidden
	}
	var out FinishSessionOutput
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		s, err := uc.Sessions.ByID(ctx, in.SessionID)
		if err != nil {
			return err
		}
		if err := s.Authorize(in.UserID); err != nil {
			return err
		}
		now := uc.Clock.Now(ctx)
		summary, err := s.Finish(now)
		if err != nil {
			return err
		}
		if err := uc.Sessions.Save(ctx, s); err != nil {
			return err
		}
		completed := *s.CompletedAt()
		out = FinishSessionOutput{
			SessionID:          s.ID(),
			Mode:               string(s.Mode()),
			CountDontKnow:      summary.CountDontKnow,
			CountStillLearning: summary.CountStillLearning,
			CountKnowKnow:      summary.CountKnowKnow,
			CompletedAt:        completed,
		}
		return uc.Events.Publish(ctx, practice.PracticeSessionCompleted{
			SessionID: s.ID(), UserID: s.UserID(), DeckID: s.DeckID(),
			Mode:               string(s.Mode()),
			CountDontKnow:      summary.CountDontKnow,
			CountStillLearning: summary.CountStillLearning,
			CountKnowKnow:      summary.CountKnowKnow,
			RatedCards:         ratedCardSummariesFrom(s.RatedCards()),
			CompletedAt:        completed,
		})
	})
	if err != nil {
		return FinishSessionOutput{}, err
	}
	return out, nil
}
