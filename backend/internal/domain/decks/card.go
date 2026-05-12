package decks

import (
	"strings"
	"unicode/utf8"
)

// Term is a 1..512-grapheme value object.
type Term struct{ value string }

// NewTerm validates and trims.
func NewTerm(raw string) (Term, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Term{}, ErrInvalidTerm
	}
	if utf8.RuneCountInString(trimmed) > MaxTermLen {
		return Term{}, ErrInvalidTerm
	}
	return Term{value: trimmed}, nil
}

// String returns the underlying value.
func (t Term) String() string { return t.value }

// Definition is a 1..2048-grapheme value object.
type Definition struct{ value string }

// NewDefinition validates and trims.
func NewDefinition(raw string) (Definition, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Definition{}, ErrInvalidDefinition
	}
	if utf8.RuneCountInString(trimmed) > MaxDefinitionLen {
		return Definition{}, ErrInvalidDefinition
	}
	return Definition{value: trimmed}, nil
}

// String returns the underlying value.
func (d Definition) String() string { return d.value }

// Card is an entity inside the Deck aggregate. It is mutated only through Deck
// methods.
type Card struct {
	id         string
	term       Term
	definition Definition
	ordinal    int
}

// ID accessor.
func (c Card) ID() string { return c.id }

// Term accessor.
func (c Card) Term() Term { return c.term }

// Definition accessor.
func (c Card) Definition() Definition { return c.definition }

// Ordinal accessor — 1-based.
func (c Card) Ordinal() int { return c.ordinal }
