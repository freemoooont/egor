// Package shared holds tiny cross-context primitives used by every domain
// package: id generation, clock, and the idempotency key value object.
package shared

import "context"

// IDGenerator mints fresh aggregate identifiers. Implementations live in
// internal/infrastructure/idgen. Domain code never imports an implementation.
type IDGenerator interface {
	NewID(ctx context.Context) string
}
