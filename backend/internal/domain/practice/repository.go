package practice

import "context"

// Sessions is the persistence port for the Session aggregate.
type Sessions interface {
	Save(ctx context.Context, s *Session) error
	ByID(ctx context.Context, id string) (*Session, error)
	LatestCompletedFor(ctx context.Context, userID, deckID string) (*Session, error)
}

// UserDeckProgresses is the persistence port for the read-model aggregate.
type UserDeckProgresses interface {
	ByUserAndDeck(ctx context.Context, userID, deckID string) (*UserDeckProgress, error)
	Save(ctx context.Context, p *UserDeckProgress) error
	DeleteByDeck(ctx context.Context, deckID string) error
}
