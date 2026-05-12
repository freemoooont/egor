// Package events provides an in-process synchronous event publisher (ADR
// 0002). It is generic over event type via the small Event interface — every
// domain event in iam, decks, and practice satisfies it. Layer 3 wires
// concrete subscribers; this package exposes the bus + a small adapter that
// also writes to the per-context outbox in the same tx.
package events

import (
	"context"
	"sync"
)

// Event is the marker interface every domain event implements. (Each
// bounded context defines its own Event interface with the same shape; we
// rely on Go's structural typing only at the top level.)
type Event interface {
	Name() string
}

// Handler is a registered subscriber for a single event name.
type Handler func(ctx context.Context, ev Event) error

// Publisher fans out events to registered handlers. Handlers run in
// registration order, in the publishing goroutine; if one returns an error,
// the rest are skipped and the error bubbles up.
type Publisher struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewPublisher builds a fresh Publisher.
func NewPublisher() *Publisher {
	return &Publisher{handlers: map[string][]Handler{}}
}

// Subscribe registers a handler for the given event name (Event.Name()).
// Multiple handlers per name run in registration order.
func (p *Publisher) Subscribe(eventName string, h Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[eventName] = append(p.handlers[eventName], h)
}

// Publish dispatches the events to every registered handler. Returns the
// first error (and stops the chain there).
func (p *Publisher) Publish(ctx context.Context, evs ...Event) error {
	p.mu.RLock()
	snapshot := make(map[string][]Handler, len(p.handlers))
	for k, v := range p.handlers {
		c := make([]Handler, len(v))
		copy(c, v)
		snapshot[k] = c
	}
	p.mu.RUnlock()
	for _, ev := range evs {
		for _, h := range snapshot[ev.Name()] {
			if err := h(ctx, ev); err != nil {
				return err
			}
		}
	}
	return nil
}
