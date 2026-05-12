package practice

import "time"

// Event marker.
type Event interface {
	Name() string
}

// PracticeSessionStarted is emitted on StartPracticeSession.
type PracticeSessionStarted struct {
	SessionID string
	UserID    string
	DeckID    string
	Mode      string
	CardIDs   []string
	StartedAt time.Time
}

// Name returns the wire name.
func (PracticeSessionStarted) Name() string { return "practice.PracticeSessionStarted" }

// CardRated is emitted on RateCard.
type CardRated struct {
	SessionID string
	UserID    string
	DeckID    string
	CardID    string
	Rating    int16
	RatedAt   time.Time
}

// Name returns the wire name.
func (CardRated) Name() string { return "practice.CardRated" }

// PracticeSessionCompleted is emitted on FinishPracticeSession.
type PracticeSessionCompleted struct {
	SessionID          string
	UserID             string
	DeckID             string
	Mode               string
	CountDontKnow      int
	CountStillLearning int
	CountKnowKnow      int
	RatedCards         []RatedCardSummary
	CompletedAt        time.Time
}

// RatedCardSummary mirrors the trimmed (CardID, Rating) shape carried in the
// PracticeSessionCompleted event payload.
type RatedCardSummary struct {
	CardID string
	Rating int16
}

// Name returns the wire name.
func (PracticeSessionCompleted) Name() string { return "practice.PracticeSessionCompleted" }
