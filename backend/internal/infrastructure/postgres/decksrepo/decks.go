// Package decksrepo implements the decks-domain Repositories using pgx/v5.
//
// A Deck aggregate is "deck row + N card rows". Save persists both atomically
// inside the active pgx.Tx (per the UnitOfWork contract): the deck row is
// upserted, then cards are reconciled in two phases — drop then re-insert —
// to keep ordinals consistent (the unique (deck_id, ordinal) is DEFERRABLE
// INITIALLY DEFERRED so reordering inside one tx is OK).
//
// Note on soft-delete timestamps: layer 1's Deck only exposes IsDeleted() (no
// DeletedAt accessor). When IsDeleted() is true on Save we set deleted_at to
// COALESCE(existing, now()) inside the SQL — preserving an earlier soft-delete
// timestamp on idempotent re-saves and stamping a fresh one on first delete.
// See .agent/tasks/micocards-mvp/raw/layer1-port-fix-request.md for the
// suggested layer-1 fix.
package decksrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/micocards/api/internal/domain/decks"
	pg "github.com/micocards/api/internal/infrastructure/postgres"
)

// Decks is the decks.Decks repository.
type Decks struct {
	pool *pgxpool.Pool
}

// NewDecks builds a Decks repository.
func NewDecks(pool *pgxpool.Pool) *Decks { return &Decks{pool: pool} }

// Save upserts the deck row and rewrites its card list. Must run inside a tx
// (the application opens it via UnitOfWork.Do); without a tx the per-row
// reconciliation can race with concurrent writes.
func (r *Decks) Save(ctx context.Context, d *decks.Deck) error {
	q := pg.Conn(ctx, r.pool)
	// Soft-delete: when IsDeleted() is true we want deleted_at non-null.
	// Use COALESCE(existing, now()) inside the UPDATE branch via an explicit
	// CASE so the timestamp isn't bumped on subsequent saves.
	if d.IsDeleted() {
		if _, err := q.Exec(ctx, `
INSERT INTO decks.decks (id, owner_id, title, created_at, deleted_at, updated_at)
VALUES ($1, $2, $3, $4, now(), now())
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    deleted_at = COALESCE(decks.decks.deleted_at, now()),
    updated_at = now()
`, d.ID(), d.OwnerID(), d.Title().String(), d.CreatedAt()); err != nil {
			return fmt.Errorf("decksrepo.Decks.Save deck (deleted): %w", err)
		}
	} else {
		if _, err := q.Exec(ctx, `
INSERT INTO decks.decks (id, owner_id, title, created_at, deleted_at, updated_at)
VALUES ($1, $2, $3, $4, NULL, now())
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    deleted_at = NULL,
    updated_at = now()
`, d.ID(), d.OwnerID(), d.Title().String(), d.CreatedAt()); err != nil {
			return fmt.Errorf("decksrepo.Decks.Save deck: %w", err)
		}
	}
	// Reconcile cards: drop all existing rows for this deck and re-insert.
	// Acceptable in v1 (decks are bounded at 500 cards by invariant 3) and
	// keeps the code simple.
	if _, err := q.Exec(ctx, `DELETE FROM decks.cards WHERE deck_id = $1`, d.ID()); err != nil {
		return fmt.Errorf("decksrepo.Decks.Save cards reset: %w", err)
	}
	for _, c := range d.Cards() {
		if _, err := q.Exec(ctx, `
INSERT INTO decks.cards (id, deck_id, ordinal, term, definition)
VALUES ($1, $2, $3, $4, $5)
`, c.ID(), d.ID(), c.Ordinal(), c.Term().String(), c.Definition().String()); err != nil {
			return fmt.Errorf("decksrepo.Decks.Save card: %w", err)
		}
	}
	return nil
}

// ByID loads a deck and all its cards.
func (r *Decks) ByID(ctx context.Context, deckID string) (*decks.Deck, error) {
	q := pg.Conn(ctx, r.pool)
	row := q.QueryRow(ctx, `
SELECT id, owner_id, title, created_at, deleted_at
FROM decks.decks
WHERE id = $1
`, deckID)
	d, err := scanDeckRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, decks.ErrDeckNotFound
		}
		return nil, fmt.Errorf("decksrepo.Decks.ByID deck: %w", err)
	}
	cards, err := r.cardsForDeck(ctx, deckID)
	if err != nil {
		return nil, err
	}
	return hydrateDeckFromRows(d, cards)
}

// ByOwner returns up to limit decks for the given owner. The cursor is
// reserved for future use (offset/keyset paging); v1 returns "" as the next
// cursor.
func (r *Decks) ByOwner(ctx context.Context, ownerID string, limit int, _ string) ([]*decks.Deck, string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := pg.Conn(ctx, r.pool)
	rows, err := q.Query(ctx, `
SELECT id, owner_id, title, created_at, deleted_at
FROM decks.decks
WHERE owner_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $2
`, ownerID, limit)
	if err != nil {
		return nil, "", fmt.Errorf("decksrepo.Decks.ByOwner: %w", err)
	}
	defer rows.Close()
	var bare []rawDeck
	for rows.Next() {
		d, err := scanDeckRow(rows)
		if err != nil {
			return nil, "", err
		}
		bare = append(bare, d)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	out := make([]*decks.Deck, 0, len(bare))
	for _, d := range bare {
		cards, err := r.cardsForDeck(ctx, d.id)
		if err != nil {
			return nil, "", err
		}
		deck, err := hydrateDeckFromRows(d, cards)
		if err != nil {
			return nil, "", err
		}
		out = append(out, deck)
	}
	return out, "", nil
}

// Delete hard-removes the deck row. The application uses Deck.Delete (soft
// delete via Save) for the lifecycle path; this is for admin/test cleanup.
func (r *Decks) Delete(ctx context.Context, deckID string) error {
	if _, err := pg.Conn(ctx, r.pool).Exec(ctx, `DELETE FROM decks.decks WHERE id = $1`, deckID); err != nil {
		return fmt.Errorf("decksrepo.Decks.Delete: %w", err)
	}
	return nil
}

// OwnerAndCards is the practice-context DeckSnapshot port. It returns the
// owner id and the start-time card-id list, in canonical ordinal order.
func (r *Decks) OwnerAndCards(ctx context.Context, deckID string) (string, []string, error) {
	q := pg.Conn(ctx, r.pool)
	var (
		ownerID   string
		deletedAt *time.Time
	)
	err := q.QueryRow(ctx, `
SELECT owner_id, deleted_at FROM decks.decks WHERE id = $1
`, deckID).Scan(&ownerID, &deletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, decks.ErrDeckNotFound
		}
		return "", nil, fmt.Errorf("decksrepo.Decks.OwnerAndCards: %w", err)
	}
	if deletedAt != nil {
		return "", nil, decks.ErrDeckNotFound
	}
	rows, err := q.Query(ctx, `
SELECT id FROM decks.cards WHERE deck_id = $1 ORDER BY ordinal ASC
`, deckID)
	if err != nil {
		return "", nil, fmt.Errorf("decksrepo.Decks.OwnerAndCards cards: %w", err)
	}
	defer rows.Close()
	var cardIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", nil, err
		}
		cardIDs = append(cardIDs, id)
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}
	return ownerID, cardIDs, nil
}

// cardsForDeck loads the cards for a deck, sorted by ordinal asc, as rawCard
// intermediates. The caller funnels them through hydrateDeckFromRows.
func (r *Decks) cardsForDeck(ctx context.Context, deckID string) ([]rawCard, error) {
	rows, err := pg.Conn(ctx, r.pool).Query(ctx, `
SELECT id, ordinal, term, definition
FROM decks.cards
WHERE deck_id = $1
ORDER BY ordinal ASC
`, deckID)
	if err != nil {
		return nil, fmt.Errorf("decksrepo.cardsForDeck: %w", err)
	}
	defer rows.Close()
	var out []rawCard
	for rows.Next() {
		c, err := scanRawCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
