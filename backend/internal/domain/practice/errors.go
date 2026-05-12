// Package practice implements the practice bounded context: sessions, ratings,
// and the user-deck progress read model. Pure domain — no infra imports.
package practice

import "errors"

// Domain sentinel errors. Mapped to HTTP via ADR 0006.
var (
	ErrSessionNotFound     = errors.New("practice: session not found")
	ErrSessionClosed       = errors.New("practice: session closed")
	ErrSessionUntracked    = errors.New("practice: session untracked")
	ErrSessionNotCompleted = errors.New("practice: session not completed")
	ErrCardNotInSession    = errors.New("practice: card not in session")
	ErrInvalidRating       = errors.New("practice: invalid rating")
	ErrInvalidPracticeMode = errors.New("practice: invalid practice mode")
	ErrForbidden           = errors.New("practice: forbidden")
	ErrDeckEmpty           = errors.New("practice: deck has no cards")
)
