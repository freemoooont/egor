package shared

import (
	"context"
	"time"
)

// Clock returns the current UTC time. Tests inject a deterministic clock so the
// domain never reaches for time.Now() directly.
type Clock interface {
	Now(ctx context.Context) time.Time
}
