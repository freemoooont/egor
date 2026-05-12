// Package decks implements the deck-authoring bounded context: decks, cards,
// and the AI generation stub. Pure domain — no infra imports.
package decks

import "errors"

// Domain sentinel errors. Mapped to HTTP via ADR 0006.
var (
	ErrInvalidDeckTitle      = errors.New("decks: invalid deck title")
	ErrDeckTitleTooLong      = errors.New("decks: deck title too long")
	ErrDeckCardLimitExceeded = errors.New("decks: deck card limit exceeded")
	ErrInvalidTerm           = errors.New("decks: invalid term")
	ErrInvalidDefinition     = errors.New("decks: invalid definition")
	ErrInvalidCardReorder    = errors.New("decks: invalid card reorder")
	ErrDeckNotFound          = errors.New("decks: deck not found")
	ErrCardNotFound          = errors.New("decks: card not found")
	ErrDeckDeleted           = errors.New("decks: deck deleted")
	ErrForbidden             = errors.New("decks: forbidden")
	ErrAINotConfigured       = errors.New("decks: ai not configured")
	ErrAIUpstream            = errors.New("decks: ai upstream error")
	ErrNotImplemented        = errors.New("decks: not implemented")
	ErrDeckEmpty             = errors.New("decks: deck has no cards")
)

// MaxCardsPerDeck is the hard cap from invariant 3.
const MaxCardsPerDeck = 500

// MaxTitleLen is the hard cap from invariant 1.
const MaxTitleLen = 120

// MaxTermLen is the hard cap from invariant 6.
const MaxTermLen = 512

// MaxDefinitionLen is the hard cap from invariant 7.
const MaxDefinitionLen = 2048
