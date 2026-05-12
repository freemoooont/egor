package practice

import (
	"context"

	"github.com/micocards/api/internal/domain/practice"
)

// GetResults reads the results of a finished session.
type GetResults struct {
	Sessions practice.Sessions
}

// Handle authorises the read and returns the rating summary.
func (uc GetResults) Handle(ctx context.Context, in GetResultsInput) (GetResultsOutput, error) {
	if in.UserID == "" {
		return GetResultsOutput{}, practice.ErrForbidden
	}
	s, err := uc.Sessions.ByID(ctx, in.SessionID)
	if err != nil {
		return GetResultsOutput{}, err
	}
	if err := s.Authorize(in.UserID); err != nil {
		return GetResultsOutput{}, err
	}
	if s.Status() != practice.StatusCompleted {
		return GetResultsOutput{}, practice.ErrSessionNotCompleted
	}
	summary := s.Summary()
	rated := s.RatedCards()
	view := make([]RatedCardView, 0, len(rated))
	for _, r := range rated {
		view = append(view, RatedCardView{CardID: r.CardID, Rating: int16(r.Rating)})
	}
	completed := s.CompletedAt()
	out := GetResultsOutput{
		SessionID: s.ID(), DeckID: s.DeckID(), Mode: string(s.Mode()),
		CountDontKnow: summary.CountDontKnow, CountStillLearning: summary.CountStillLearning, CountKnowKnow: summary.CountKnowKnow,
		RatedCards: view,
	}
	if completed != nil {
		out.CompletedAt = *completed
	}
	return out, nil
}
