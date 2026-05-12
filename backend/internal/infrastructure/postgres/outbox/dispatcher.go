package outbox

import (
	"context"
	"fmt"
)

// EventForwarder is the minimal port the dispatcher calls — typically the
// in-process events.Publisher (`internal/infrastructure/events`).
type EventForwarder interface {
	Forward(ctx context.Context, eventName string, payload []byte, idempotencyKey string) error
}

// Dispatcher pulls undispatched rows out of the outbox and forwards them to a
// sync EventForwarder. Layer 3 wires this into a janitor goroutine that ticks
// every N seconds; the call is also useful as a one-shot from tests.
type Dispatcher struct {
	box       *Outbox
	forwarder EventForwarder
	batchSize int
}

// NewDispatcher builds a Dispatcher. batchSize <= 0 falls back to 100.
func NewDispatcher(box *Outbox, fwd EventForwarder, batchSize int) *Dispatcher {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &Dispatcher{box: box, forwarder: fwd, batchSize: batchSize}
}

// DispatchOnce drains one batch of undispatched rows, forwarding each. Returns
// the number successfully forwarded, plus any forwarding error (which stops
// the loop early).
func (d *Dispatcher) DispatchOnce(ctx context.Context) (int, error) {
	rows, err := d.box.FetchUndispatchedBatch(ctx, d.batchSize)
	if err != nil {
		return 0, err
	}
	for i, r := range rows {
		if err := d.forwarder.Forward(ctx, r.EventName, r.Payload, r.IdempotencyKey); err != nil {
			return i, fmt.Errorf("dispatcher: forward %s: %w", r.EventName, err)
		}
		if err := d.box.MarkDispatched(ctx, r.ID); err != nil {
			return i, fmt.Errorf("dispatcher: mark dispatched: %w", err)
		}
	}
	return len(rows), nil
}
