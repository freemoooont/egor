package decksrepo

import (
	"fmt"
	"time"

	"github.com/micocards/api/internal/domain/decks"
)

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// rawDeck is an intermediate representation between the SQL row and the domain
// entity — built first, fed into hydrateDeckFromRows once cards have been
// loaded.
type rawDeck struct {
	id        string
	ownerID   string
	title     decks.DeckTitle
	createdAt time.Time
	deletedAt *time.Time
}

// scanDeckRow scans the canonical (id, owner_id, title, created_at, deleted_at)
// projection into a rawDeck.
func scanDeckRow(s rowScanner) (rawDeck, error) {
	var (
		id, owner, title string
		createdAt        time.Time
		deletedAt        *time.Time
	)
	if err := s.Scan(&id, &owner, &title, &createdAt, &deletedAt); err != nil {
		return rawDeck{}, err
	}
	titleVO, err := decks.NewDeckTitle(title)
	if err != nil {
		return rawDeck{}, fmt.Errorf("scanDeckRow title: %w", err)
	}
	out := rawDeck{
		id:        id,
		ownerID:   owner,
		title:     titleVO,
		createdAt: createdAt,
	}
	if deletedAt != nil {
		t := deletedAt.UTC()
		out.deletedAt = &t
	}
	return out, nil
}

type rawCard struct {
	id      string
	ordinal int
	term    decks.Term
	def     decks.Definition
}

func scanRawCard(s rowScanner) (rawCard, error) {
	var (
		id        string
		ordinal   int
		term, def string
	)
	if err := s.Scan(&id, &ordinal, &term, &def); err != nil {
		return rawCard{}, err
	}
	t, err := decks.NewTerm(term)
	if err != nil {
		return rawCard{}, fmt.Errorf("scanRawCard term: %w", err)
	}
	d, err := decks.NewDefinition(def)
	if err != nil {
		return rawCard{}, fmt.Errorf("scanRawCard definition: %w", err)
	}
	return rawCard{id: id, ordinal: ordinal, term: t, def: d}, nil
}

// hydrateDeckFromRows assembles a domain Deck from the scanned deck row plus
// the scanned card rows (already sorted by ordinal asc). It works around layer
// 1 not exposing HydrateCard: we build a fresh Deck via NewDeck and re-feed
// each card via AddCard, which produces dense 1..N ordinals. Because the DB
// already enforces dense ordinals, the resulting deck matches the persisted
// state exactly.
func hydrateDeckFromRows(d rawDeck, cards []rawCard) (*decks.Deck, error) {
	deck, err := decks.NewDeck(d.id, d.ownerID, d.title, d.createdAt)
	if err != nil {
		return nil, fmt.Errorf("hydrateDeckFromRows deck: %w", err)
	}
	for _, c := range cards {
		if _, err := deck.AddCard(d.ownerID, c.id, c.term, c.def); err != nil {
			return nil, fmt.Errorf("hydrateDeckFromRows card %s: %w", c.id, err)
		}
	}
	if d.deletedAt != nil {
		// Restore the soft-deleted flag at the persisted timestamp.
		if err := deck.Delete(d.ownerID, *d.deletedAt); err != nil {
			return nil, fmt.Errorf("hydrateDeckFromRows soft-delete: %w", err)
		}
	}
	return deck, nil
}

