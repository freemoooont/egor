// Package clock provides Clock implementations.
//
// Real wall-clock impl in [System]; deterministic in [Fixed]. The application
// layer should always depend on the [shared.Clock] port, never on this package
// directly outside of cmd/api wiring (or test files).
package clock

import (
	"context"
	"sync"
	"time"
)

// System reads the wall clock; UTC by construction.
type System struct{}

// Now returns time.Now in UTC.
func (System) Now(_ context.Context) time.Time { return time.Now().UTC() }

// Fixed is a deterministic clock used in tests.
type Fixed struct {
	mu sync.Mutex
	t  time.Time
}

// NewFixed creates a Fixed clock at the supplied moment (UTC).
func NewFixed(at time.Time) *Fixed {
	return &Fixed{t: at.UTC()}
}

// Now returns the current pinned time.
func (f *Fixed) Now(_ context.Context) time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

// Advance moves the clock forward by d.
func (f *Fixed) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

// Set pins the clock to a specific moment.
func (f *Fixed) Set(at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = at.UTC()
}
