// Package practice contains the HTTP DTOs for the practice context.
package practice

import "time"

// StartPracticeRequest is the body for POST /api/practice/sessions.
type StartPracticeRequest struct {
	DeckID string `json:"deckId"`
	Mode   string `json:"mode"`
}

// PracticeSession mirrors the OpenAPI shape.
type PracticeSession struct {
	ID          string     `json:"id"`
	DeckID      string     `json:"deckId"`
	UserID      string     `json:"userId"`
	Mode        string     `json:"mode"`
	Status      string     `json:"status"`
	CardIDs     []string   `json:"cardIds"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// RateCardRequest is the body for POST /api/practice/sessions/{id}/ratings.
type RateCardRequest struct {
	CardID string `json:"cardId"`
	Rating int16  `json:"rating"`
}

// RatedCard mirrors the OpenAPI shape.
type RatedCard struct {
	SessionID string    `json:"sessionId"`
	CardID    string    `json:"cardId"`
	Rating    int16     `json:"rating"`
	RatedAt   time.Time `json:"ratedAt"`
}

// PracticeResults mirrors the OpenAPI shape.
type PracticeResults struct {
	SessionID          string      `json:"sessionId"`
	DeckID             string      `json:"deckId"`
	Mode               string      `json:"mode"`
	CountDontKnow      int         `json:"countDontKnow"`
	CountStillLearning int         `json:"countStillLearning"`
	CountKnowKnow      int         `json:"countKnowKnow"`
	RatedCards         []RatedCard `json:"ratedCards"`
	CompletedAt        time.Time   `json:"completedAt"`
}

// CardProgress is one card's spaced-repetition progress entry.
type CardProgress struct {
	CardID      string    `json:"cardId"`
	Rating      int16     `json:"rating"`
	LastRatedAt time.Time `json:"lastRatedAt"`
}

// UserDeckProgress mirrors the OpenAPI shape.
type UserDeckProgress struct {
	DeckID       string         `json:"deckId"`
	CardProgress []CardProgress `json:"cardProgress"`
}
