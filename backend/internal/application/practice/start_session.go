package practice

import (
	"context"
	"errors"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/domain/shared"
)

// StartSession opens a new practice session for a deck.
type StartSession struct {
	Sessions  practice.Sessions
	Snapshots DeckSnapshot
	IDs       shared.IDGenerator
	Clock     shared.Clock
	UoW       UnitOfWork
	Events    EventPublisher
}

// Handle authorises and creates a new in-progress session.
func (uc StartSession) Handle(ctx context.Context, in StartSessionInput) (StartSessionOutput, error) {
	if in.UserID == "" {
		return StartSessionOutput{}, practice.ErrForbidden
	}
	mode := practice.SessionMode(in.Mode)
	if !mode.IsValid() {
		return StartSessionOutput{}, practice.ErrInvalidPracticeMode
	}

	var out StartSessionOutput
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		ownerID, cardIDs, err := uc.Snapshots.OwnerAndCards(ctx, in.DeckID)
		if err != nil {
			if errors.Is(err, decks.ErrDeckNotFound) {
				return decks.ErrDeckNotFound
			}
			return err
		}
		if ownerID != in.UserID {
			return practice.ErrForbidden
		}
		if len(cardIDs) == 0 {
			return practice.ErrDeckEmpty
		}

		now := uc.Clock.Now(ctx)
		sid := uc.IDs.NewID(ctx)
		s, err := practice.NewSession(sid, in.UserID, in.DeckID, mode, cardIDs, now)
		if err != nil {
			return err
		}
		if err := uc.Sessions.Save(ctx, s); err != nil {
			return err
		}
		out = StartSessionOutput{
			SessionID: s.ID(), DeckID: s.DeckID(), Mode: string(s.Mode()),
			CardIDs: s.CardIDs(), StartedAt: s.StartedAt(),
		}
		return uc.Events.Publish(ctx, practice.PracticeSessionStarted{
			SessionID: s.ID(), UserID: s.UserID(), DeckID: s.DeckID(),
			Mode: string(s.Mode()), CardIDs: s.CardIDs(), StartedAt: s.StartedAt(),
		})
	})
	if err != nil {
		return StartSessionOutput{}, err
	}
	return out, nil
}
