// Package decks contains the decks-context use cases.
package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
)

// EventPublisher dispatches decks events on the in-process bus.
type EventPublisher interface {
	Publish(ctx context.Context, events ...decks.Event) error
}

// Outbox is the persistence port for the decks outbox table (ADR 0002).
type Outbox interface {
	Append(ctx context.Context, eventName string, payload []byte, idempotencyKey string) error
}

// UnitOfWork runs the supplied callback inside a single pgx.Tx.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// AIProvider is the port for the AI deck generation stub. Implementations
// either return ErrAINotConfigured (default in v1) or an actual draft.
type AIProvider interface {
	IsConfigured() bool
	Generate(ctx context.Context, prompt string) (decks.AIDeckDraft, error)
}
