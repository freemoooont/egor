// Package practice contains the practice-context use cases.
package practice

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/practice"
)

// EventPublisher dispatches practice events on the in-process bus.
type EventPublisher interface {
	Publish(ctx context.Context, events ...practice.Event) error
}

// Outbox is the persistence port for the practice outbox table (ADR 0002).
type Outbox interface {
	Append(ctx context.Context, eventName string, payload []byte, idempotencyKey string) error
}

// UnitOfWork runs the supplied callback inside a single pgx.Tx.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// DeckSnapshot is the read-only port practice uses to fetch the deck's card
// ids at session start. Implementations live alongside the decks repo.
type DeckSnapshot interface {
	OwnerAndCards(ctx context.Context, deckID string) (ownerID string, cardIDs []string, err error)
}

// AnswerForDeckSnapshotMissing maps the decks-domain not-found into practice-domain.
var (
	_ = decks.ErrDeckNotFound // referenced for documentation; the snapshot impl returns it.
)
