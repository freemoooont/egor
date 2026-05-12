package events

import (
	"context"
	"encoding/json"

	"github.com/micocards/api/internal/domain/decks"
	domiam "github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/domain/practice"
)

// OutboxWriter is the small interface every per-context outbox implementation
// satisfies. The infra outbox.Outbox already satisfies it.
type OutboxWriter interface {
	Append(ctx context.Context, eventName string, payload []byte, idempotencyKey string) error
}

// IAMPublisher adapts the generic Publisher to the iam application port.
// Publish writes the event to the iam outbox first (so durability is part of
// the use-case tx), then dispatches synchronously.
type IAMPublisher struct {
	bus    *Publisher
	outbox OutboxWriter
}

// NewIAMPublisher wires the bus + outbox.
func NewIAMPublisher(bus *Publisher, outbox OutboxWriter) *IAMPublisher {
	return &IAMPublisher{bus: bus, outbox: outbox}
}

// Publish appends each event to the outbox and dispatches in-process.
func (p *IAMPublisher) Publish(ctx context.Context, events ...domiam.Event) error {
	for _, ev := range events {
		payload, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if p.outbox != nil {
			if err := p.outbox.Append(ctx, ev.Name(), payload, ""); err != nil {
				return err
			}
		}
		if p.bus != nil {
			if err := p.bus.Publish(ctx, ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// DecksPublisher adapts the generic Publisher to the decks application port.
type DecksPublisher struct {
	bus    *Publisher
	outbox OutboxWriter
}

// NewDecksPublisher wires the bus + outbox.
func NewDecksPublisher(bus *Publisher, outbox OutboxWriter) *DecksPublisher {
	return &DecksPublisher{bus: bus, outbox: outbox}
}

// Publish appends each event to the outbox and dispatches in-process.
func (p *DecksPublisher) Publish(ctx context.Context, events ...decks.Event) error {
	for _, ev := range events {
		payload, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if p.outbox != nil {
			if err := p.outbox.Append(ctx, ev.Name(), payload, ""); err != nil {
				return err
			}
		}
		if p.bus != nil {
			if err := p.bus.Publish(ctx, ev); err != nil {
				return err
			}
		}
	}
	return nil
}

// PracticePublisher adapts the generic Publisher to the practice application port.
type PracticePublisher struct {
	bus    *Publisher
	outbox OutboxWriter
}

// NewPracticePublisher wires the bus + outbox.
func NewPracticePublisher(bus *Publisher, outbox OutboxWriter) *PracticePublisher {
	return &PracticePublisher{bus: bus, outbox: outbox}
}

// Publish appends each event to the outbox and dispatches in-process.
func (p *PracticePublisher) Publish(ctx context.Context, events ...practice.Event) error {
	for _, ev := range events {
		payload, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if p.outbox != nil {
			if err := p.outbox.Append(ctx, ev.Name(), payload, ""); err != nil {
				return err
			}
		}
		if p.bus != nil {
			if err := p.bus.Publish(ctx, ev); err != nil {
				return err
			}
		}
	}
	return nil
}
