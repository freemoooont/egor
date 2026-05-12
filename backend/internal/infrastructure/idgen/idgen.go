// Package idgen provides IDGenerator implementations.
//
// UUID is the production impl; Sequential is a deterministic test fake.
package idgen

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// UUID generates UUID v4 ids.
type UUID struct{}

// NewID returns a UUID v4 string.
func (UUID) NewID(_ context.Context) string { return uuid.NewString() }

// Sequential is a deterministic id generator used in tests.
type Sequential struct {
	prefix string
	n      atomic.Int64
}

// NewSequential builds a Sequential generator emitting "<prefix>-<n>" ids.
func NewSequential(prefix string) *Sequential { return &Sequential{prefix: prefix} }

// NewID returns the next id in sequence.
func (s *Sequential) NewID(_ context.Context) string {
	v := s.n.Add(1)
	return fmt.Sprintf("%s-%d", s.prefix, v)
}

// Static is a fixed-id generator used by use-case tests that need a known id.
type Static struct {
	mu  sync.Mutex
	ids []string
}

// NewStatic builds a Static generator that returns ids in order; subsequent
// calls past len(ids) return ids[last].
func NewStatic(ids ...string) *Static { return &Static{ids: append([]string(nil), ids...)} }

// NewID returns the next pre-supplied id.
func (s *Static) NewID(_ context.Context) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.ids) == 0 {
		return ""
	}
	if len(s.ids) == 1 {
		return s.ids[0]
	}
	v := s.ids[0]
	s.ids = s.ids[1:]
	return v
}
