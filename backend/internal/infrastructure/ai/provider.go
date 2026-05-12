// Package ai implements the decks-application AIProvider port. v1 ships only
// the disabled stub — the openai.Stub type documents the expected request
// shape so layer 3 (or v2) can swap a real adapter in.
package ai

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
)

// NotConfigured is the default impl: IsConfigured() == false; Generate
// returns ErrAINotConfigured. Wire it whenever AI_API_KEY is empty.
type NotConfigured struct{}

// IsConfigured returns false.
func (NotConfigured) IsConfigured() bool { return false }

// Generate returns the sentinel from the decks domain so the use case maps it
// to HTTP 501 cleanly.
func (NotConfigured) Generate(_ context.Context, _ string) (decks.AIDeckDraft, error) {
	return decks.AIDeckDraft{}, decks.ErrAINotConfigured
}
