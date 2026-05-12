package decks

import (
	"strings"
	"time"
	"unicode/utf8"
)

// DeckTitle is a 1..120-grapheme value object.
type DeckTitle struct{ value string }

// NewDeckTitle validates and trims.
//
// Invariant: Deck_TitleLengthBetween1And120.
func NewDeckTitle(raw string) (DeckTitle, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DeckTitle{}, ErrInvalidDeckTitle
	}
	if utf8.RuneCountInString(trimmed) > MaxTitleLen {
		return DeckTitle{}, ErrDeckTitleTooLong
	}
	return DeckTitle{value: trimmed}, nil
}

// String returns the underlying value.
func (t DeckTitle) String() string { return t.value }

// Deck is the aggregate root. cards are mutated only through Deck methods.
type Deck struct {
	id        string
	ownerID   string
	title     DeckTitle
	cards     []Card
	createdAt time.Time
	deletedAt *time.Time
}

// NewDeck builds a fresh deck. ownerID is immutable (invariant 2).
func NewDeck(id, ownerID string, title DeckTitle, createdAt time.Time) (*Deck, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(ownerID) == "" {
		return nil, ErrDeckNotFound
	}
	if title.String() == "" {
		return nil, ErrInvalidDeckTitle
	}
	return &Deck{
		id:        id,
		ownerID:   ownerID,
		title:     title,
		createdAt: createdAt.UTC(),
	}, nil
}

// HydrateDeck rebuilds a Deck from persistence.
func HydrateDeck(id, ownerID string, title DeckTitle, cards []Card, createdAt time.Time, deletedAt *time.Time) *Deck {
	out := &Deck{
		id:        id,
		ownerID:   ownerID,
		title:     title,
		cards:     append([]Card(nil), cards...),
		createdAt: createdAt.UTC(),
	}
	if deletedAt != nil {
		v := deletedAt.UTC()
		out.deletedAt = &v
	}
	return out
}

// ID accessor.
func (d *Deck) ID() string { return d.id }

// OwnerID accessor.
func (d *Deck) OwnerID() string { return d.ownerID }

// Title accessor.
func (d *Deck) Title() DeckTitle { return d.title }

// Cards returns a defensive copy of the card list.
func (d *Deck) Cards() []Card {
	out := make([]Card, len(d.cards))
	copy(out, d.cards)
	return out
}

// CardCount returns the current number of cards.
func (d *Deck) CardCount() int { return len(d.cards) }

// CreatedAt accessor.
func (d *Deck) CreatedAt() time.Time { return d.createdAt }

// IsDeleted reports whether the deck has been soft-deleted.
func (d *Deck) IsDeleted() bool { return d.deletedAt != nil }

// authorize checks invariant 13.
func (d *Deck) authorize(callerID string) error {
	if callerID != d.ownerID {
		return ErrForbidden
	}
	return nil
}

// requireAlive checks invariant 12.
func (d *Deck) requireAlive() error {
	if d.IsDeleted() {
		return ErrDeckDeleted
	}
	return nil
}

// Rename mutates the title. Invariants 1, 11, 12, 13.
func (d *Deck) Rename(callerID string, newTitle DeckTitle) error {
	if err := d.authorize(callerID); err != nil {
		return err
	}
	if err := d.requireAlive(); err != nil {
		return err
	}
	if newTitle.String() == "" {
		return ErrInvalidDeckTitle
	}
	d.title = newTitle
	return nil
}

// AddCard appends a card with the next ordinal. Invariants 3, 4, 5, 8, 13.
func (d *Deck) AddCard(callerID, cardID string, term Term, def Definition) (Card, error) {
	if err := d.authorize(callerID); err != nil {
		return Card{}, err
	}
	if err := d.requireAlive(); err != nil {
		return Card{}, err
	}
	if len(d.cards) >= MaxCardsPerDeck {
		return Card{}, ErrDeckCardLimitExceeded
	}
	if strings.TrimSpace(cardID) == "" {
		return Card{}, ErrCardNotFound
	}
	c := Card{id: cardID, term: term, definition: def, ordinal: len(d.cards) + 1}
	d.cards = append(d.cards, c)
	return c, nil
}

// EditCard mutates term/definition. Invariants 6, 7, 13.
func (d *Deck) EditCard(callerID, cardID string, newTerm *Term, newDef *Definition) (Card, error) {
	if err := d.authorize(callerID); err != nil {
		return Card{}, err
	}
	if err := d.requireAlive(); err != nil {
		return Card{}, err
	}
	idx := d.indexOfCard(cardID)
	if idx < 0 {
		return Card{}, ErrCardNotFound
	}
	if newTerm != nil {
		d.cards[idx].term = *newTerm
	}
	if newDef != nil {
		d.cards[idx].definition = *newDef
	}
	return d.cards[idx], nil
}

// RemoveCard removes the card and re-densifies ordinals. Invariants 9, 13.
func (d *Deck) RemoveCard(callerID, cardID string) error {
	if err := d.authorize(callerID); err != nil {
		return err
	}
	if err := d.requireAlive(); err != nil {
		return err
	}
	idx := d.indexOfCard(cardID)
	if idx < 0 {
		return ErrCardNotFound
	}
	d.cards = append(d.cards[:idx], d.cards[idx+1:]...)
	for i := range d.cards {
		d.cards[i].ordinal = i + 1
	}
	return nil
}

// ReorderCards rewrites ordinals from the supplied permutation. Invariants
// 4, 5, 10, 13.
func (d *Deck) ReorderCards(callerID string, orderedIDs []string) error {
	if err := d.authorize(callerID); err != nil {
		return err
	}
	if err := d.requireAlive(); err != nil {
		return err
	}
	if len(orderedIDs) != len(d.cards) {
		return ErrInvalidCardReorder
	}
	byID := make(map[string]Card, len(d.cards))
	for _, c := range d.cards {
		byID[c.id] = c
	}
	seen := make(map[string]bool, len(orderedIDs))
	rebuilt := make([]Card, 0, len(orderedIDs))
	for i, id := range orderedIDs {
		if seen[id] {
			return ErrInvalidCardReorder
		}
		c, ok := byID[id]
		if !ok {
			return ErrInvalidCardReorder
		}
		seen[id] = true
		c.ordinal = i + 1
		rebuilt = append(rebuilt, c)
	}
	d.cards = rebuilt
	return nil
}

// Delete soft-deletes the deck (invariant 12 makes subsequent mutations error).
func (d *Deck) Delete(callerID string, at time.Time) error {
	if err := d.authorize(callerID); err != nil {
		return err
	}
	if d.IsDeleted() {
		return nil
	}
	t := at.UTC()
	d.deletedAt = &t
	return nil
}

func (d *Deck) indexOfCard(id string) int {
	for i, c := range d.cards {
		if c.id == id {
			return i
		}
	}
	return -1
}
