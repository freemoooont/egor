package practice

import (
	"strings"
	"time"
)

// RatedCard is a child entity inside Session.
type RatedCard struct {
	CardID  string
	Rating  CardRating
	RatedAt time.Time
}

// Session is the practice aggregate root.
type Session struct {
	id          string
	userID      string
	deckID      string
	mode        SessionMode
	status      SessionStatus
	cardIDs     []string
	rated       map[string]RatedCard
	startedAt   time.Time
	completedAt *time.Time
	abandonedAt *time.Time
}

// NewSession constructs a session in InProgress (invariant 1) and pins the
// owner+deck (invariants 8, 9). cardIDs is the snapshot of the deck at start.
func NewSession(id, userID, deckID string, mode SessionMode, cardIDs []string, startedAt time.Time) (*Session, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(deckID) == "" {
		return nil, ErrSessionNotFound
	}
	if !mode.IsValid() {
		return nil, ErrInvalidPracticeMode
	}
	if len(cardIDs) == 0 {
		return nil, ErrDeckEmpty
	}
	snapshot := make([]string, len(cardIDs))
	copy(snapshot, cardIDs)
	return &Session{
		id:        id,
		userID:    userID,
		deckID:    deckID,
		mode:      mode,
		status:    StatusInProgress,
		cardIDs:   snapshot,
		rated:     make(map[string]RatedCard),
		startedAt: startedAt.UTC(),
	}, nil
}

// HydrateSession rebuilds a Session from persistence.
func HydrateSession(
	id, userID, deckID string,
	mode SessionMode,
	status SessionStatus,
	cardIDs []string,
	rated []RatedCard,
	startedAt time.Time,
	completedAt, abandonedAt *time.Time,
) *Session {
	rmap := make(map[string]RatedCard, len(rated))
	for _, r := range rated {
		rmap[r.CardID] = r
	}
	out := &Session{
		id:        id,
		userID:    userID,
		deckID:    deckID,
		mode:      mode,
		status:    status,
		cardIDs:   append([]string(nil), cardIDs...),
		rated:     rmap,
		startedAt: startedAt.UTC(),
	}
	if completedAt != nil {
		t := completedAt.UTC()
		out.completedAt = &t
	}
	if abandonedAt != nil {
		t := abandonedAt.UTC()
		out.abandonedAt = &t
	}
	return out
}

// ID accessor.
func (s *Session) ID() string { return s.id }

// UserID accessor.
func (s *Session) UserID() string { return s.userID }

// DeckID accessor.
func (s *Session) DeckID() string { return s.deckID }

// Mode accessor.
func (s *Session) Mode() SessionMode { return s.mode }

// Status accessor.
func (s *Session) Status() SessionStatus { return s.status }

// CardIDs returns the start-time snapshot.
func (s *Session) CardIDs() []string {
	out := make([]string, len(s.cardIDs))
	copy(out, s.cardIDs)
	return out
}

// RatedCards returns a deterministic-ordered list of rated cards.
func (s *Session) RatedCards() []RatedCard {
	out := make([]RatedCard, 0, len(s.rated))
	for _, id := range s.cardIDs {
		if r, ok := s.rated[id]; ok {
			out = append(out, r)
		}
	}
	return out
}

// StartedAt accessor.
func (s *Session) StartedAt() time.Time { return s.startedAt }

// CompletedAt accessor.
func (s *Session) CompletedAt() *time.Time { return s.completedAt }

// AbandonedAt accessor.
func (s *Session) AbandonedAt() *time.Time { return s.abandonedAt }

// Authorize checks ownership.
func (s *Session) Authorize(userID string) error {
	if userID != s.userID {
		return ErrForbidden
	}
	return nil
}

// Rate records a rating. Invariants 2, 3, 4, 5.
func (s *Session) Rate(cardID string, rating CardRating, at time.Time) (RatedCard, error) {
	if s.status != StatusInProgress {
		return RatedCard{}, ErrSessionClosed
	}
	if s.mode != ModeTracked {
		return RatedCard{}, ErrSessionUntracked
	}
	if !rating.IsValid() {
		return RatedCard{}, ErrInvalidRating
	}
	if !s.cardInSnapshot(cardID) {
		return RatedCard{}, ErrCardNotInSession
	}
	rc := RatedCard{CardID: cardID, Rating: rating, RatedAt: at.UTC()}
	s.rated[cardID] = rc
	return rc, nil
}

// Finish closes the session (invariant 6). Idempotent: subsequent calls return
// the previously stamped completed_at.
func (s *Session) Finish(at time.Time) (Summary, error) {
	if s.status == StatusAbandoned {
		return Summary{}, ErrSessionClosed
	}
	if s.status != StatusCompleted {
		t := at.UTC()
		s.completedAt = &t
		s.status = StatusCompleted
	}
	return s.Summary(), nil
}

// Abandon closes the session as abandoned (invariant 7).
func (s *Session) Abandon(at time.Time) error {
	if s.status == StatusCompleted {
		return ErrSessionClosed
	}
	if s.status != StatusAbandoned {
		t := at.UTC()
		s.abandonedAt = &t
		s.status = StatusAbandoned
	}
	return nil
}

// Summary returns the per-rating counts. Invariant 10.
type Summary struct {
	CountDontKnow      int
	CountStillLearning int
	CountKnowKnow      int
}

// Summary computes the rating counts from the current rated map.
func (s *Session) Summary() Summary {
	var sum Summary
	for _, r := range s.rated {
		switch r.Rating {
		case RatingDontKnow:
			sum.CountDontKnow++
		case RatingStillLearning:
			sum.CountStillLearning++
		case RatingKnowKnow:
			sum.CountKnowKnow++
		}
	}
	return sum
}

func (s *Session) cardInSnapshot(cardID string) bool {
	for _, id := range s.cardIDs {
		if id == cardID {
			return true
		}
	}
	return false
}
