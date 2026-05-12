package practice

import (
	"time"

	"github.com/micocards/api/internal/domain/practice"
)

// StartSessionInput / Output.
type StartSessionInput struct {
	UserID string
	DeckID string
	Mode   string
}

// StartSessionOutput is the response payload.
type StartSessionOutput struct {
	SessionID string
	DeckID    string
	Mode      string
	CardIDs   []string
	StartedAt time.Time
}

// RateCardInput / Output.
type RateCardInput struct {
	UserID    string
	SessionID string
	CardID    string
	Rating    int16
}

// RateCardOutput is the response payload.
type RateCardOutput struct {
	SessionID string
	CardID    string
	Rating    int16
	RatedAt   time.Time
}

// FinishSessionInput / Output.
type FinishSessionInput struct {
	UserID    string
	SessionID string
}

// FinishSessionOutput is the response payload.
type FinishSessionOutput struct {
	SessionID          string
	Mode               string
	CountDontKnow      int
	CountStillLearning int
	CountKnowKnow      int
	CompletedAt        time.Time
}

// GetResultsInput / Output.
type GetResultsInput struct {
	UserID    string
	SessionID string
}

// GetResultsOutput is the response payload.
type GetResultsOutput struct {
	SessionID          string
	DeckID             string
	Mode               string
	CountDontKnow      int
	CountStillLearning int
	CountKnowKnow      int
	RatedCards         []RatedCardView
	CompletedAt        time.Time
}

// RatedCardView is the wire shape.
type RatedCardView struct {
	CardID string
	Rating int16
}

// GetUserDeckProgressInput / Output.
type GetUserDeckProgressInput struct {
	UserID string
	DeckID string
}

// CardProgressView is one entry in the progress projection.
type CardProgressView struct {
	CardID      string
	Rating      int16
	LastRatedAt time.Time
}

// GetUserDeckProgressOutput is the response payload.
type GetUserDeckProgressOutput struct {
	DeckID       string
	CardProgress []CardProgressView
}

// ratedCardSummariesFrom converts domain rated cards into the event payload shape.
func ratedCardSummariesFrom(rated []practice.RatedCard) []practice.RatedCardSummary {
	out := make([]practice.RatedCardSummary, 0, len(rated))
	for _, r := range rated {
		out = append(out, practice.RatedCardSummary{CardID: r.CardID, Rating: int16(r.Rating)})
	}
	return out
}
