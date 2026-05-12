package ai

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
)

// OpenAIStub is an interface-only documentation of the request shape the v2
// OpenAI adapter is expected to implement. The current implementation is
// intentionally identical to NotConfigured — it returns ErrAINotConfigured —
// but its zero value carries the API key so that wiring in cmd/api can pass
// the env value through without changing layer 3.
type OpenAIStub struct {
	// APIKey is the OpenAI API key. Empty means "not configured".
	APIKey string
	// Model identifier (e.g. "gpt-4o-mini"). Empty means default.
	Model string
}

// IsConfigured reports whether the adapter has enough config to make a real
// upstream call.
func (s OpenAIStub) IsConfigured() bool { return s.APIKey != "" }

// Generate is the v1 stub. Even when APIKey is set we return
// ErrAINotConfigured because the real upstream call is deferred to v2 (the
// task spec marks AI generation as out of scope).
func (s OpenAIStub) Generate(_ context.Context, _ string) (decks.AIDeckDraft, error) {
	return decks.AIDeckDraft{}, decks.ErrAINotConfigured
}
