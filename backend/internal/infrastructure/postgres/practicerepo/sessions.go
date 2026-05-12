// Package practicerepo implements the practice-domain Repositories using
// pgx/v5 against the practice.* schema. Sessions and UserDeckProgress are the
// two aggregates; ratings live as a child entity table for Sessions.
package practicerepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/micocards/api/internal/domain/practice"
	pg "github.com/micocards/api/internal/infrastructure/postgres"
)

// Sessions is the practice.Sessions repository.
type Sessions struct {
	pool *pgxpool.Pool
}

// NewSessions builds a Sessions repository.
func NewSessions(pool *pgxpool.Pool) *Sessions { return &Sessions{pool: pool} }

// Save upserts the session row and reconciles its rated_cards rows.
func (r *Sessions) Save(ctx context.Context, s *practice.Session) error {
	q := pg.Conn(ctx, r.pool)
	cardIDsJSON, err := json.Marshal(s.CardIDs())
	if err != nil {
		return fmt.Errorf("practicerepo.Sessions.Save card_ids json: %w", err)
	}
	if _, err := q.Exec(ctx, `
INSERT INTO practice.sessions (
    id, user_id, deck_id, mode, status, card_ids, started_at, completed_at, abandoned_at
) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    completed_at = EXCLUDED.completed_at,
    abandoned_at = EXCLUDED.abandoned_at
`,
		s.ID(), s.UserID(), s.DeckID(),
		string(s.Mode()), string(s.Status()),
		string(cardIDsJSON),
		s.StartedAt(), s.CompletedAt(), s.AbandonedAt(),
	); err != nil {
		return fmt.Errorf("practicerepo.Sessions.Save session: %w", err)
	}
	// Reconcile ratings: drop and re-insert. Bounded by deck size (≤500).
	if _, err := q.Exec(ctx, `DELETE FROM practice.session_card_ratings WHERE session_id = $1`, s.ID()); err != nil {
		return fmt.Errorf("practicerepo.Sessions.Save reset ratings: %w", err)
	}
	for i, rc := range s.RatedCards() {
		// Synthetic rating-row id: <session_id>:r:<index>. Stable for the
		// lifetime of the session because RatedCards() is deterministic.
		id := fmt.Sprintf("%s:r:%d", s.ID(), i)
		if _, err := q.Exec(ctx, `
INSERT INTO practice.session_card_ratings (id, session_id, card_id, rating, rated_at)
VALUES ($1, $2, $3, $4, $5)
`, id, s.ID(), rc.CardID, int16(rc.Rating), rc.RatedAt); err != nil {
			return fmt.Errorf("practicerepo.Sessions.Save rating: %w", err)
		}
	}
	return nil
}

// ByID loads a session and its ratings. Returns ErrSessionNotFound if missing.
func (r *Sessions) ByID(ctx context.Context, id string) (*practice.Session, error) {
	q := pg.Conn(ctx, r.pool)
	row := q.QueryRow(ctx, `
SELECT id, user_id, deck_id, mode, status, card_ids, started_at, completed_at, abandoned_at
FROM practice.sessions
WHERE id = $1
`, id)
	raw, err := scanSessionRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, practice.ErrSessionNotFound
		}
		return nil, fmt.Errorf("practicerepo.Sessions.ByID: %w", err)
	}
	rated, err := r.ratingsForSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return practice.HydrateSession(
		raw.id, raw.userID, raw.deckID,
		raw.mode, raw.status,
		raw.cardIDs, rated,
		raw.startedAt, raw.completedAt, raw.abandonedAt,
	), nil
}

// LatestCompletedFor returns the most recently completed session for the given
// user/deck pair, or ErrSessionNotFound if none.
func (r *Sessions) LatestCompletedFor(ctx context.Context, userID, deckID string) (*practice.Session, error) {
	row := pg.Conn(ctx, r.pool).QueryRow(ctx, `
SELECT id, user_id, deck_id, mode, status, card_ids, started_at, completed_at, abandoned_at
FROM practice.sessions
WHERE user_id = $1 AND deck_id = $2 AND completed_at IS NOT NULL
ORDER BY completed_at DESC
LIMIT 1
`, userID, deckID)
	raw, err := scanSessionRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, practice.ErrSessionNotFound
		}
		return nil, fmt.Errorf("practicerepo.Sessions.LatestCompletedFor: %w", err)
	}
	rated, err := r.ratingsForSession(ctx, raw.id)
	if err != nil {
		return nil, err
	}
	return practice.HydrateSession(
		raw.id, raw.userID, raw.deckID,
		raw.mode, raw.status,
		raw.cardIDs, rated,
		raw.startedAt, raw.completedAt, raw.abandonedAt,
	), nil
}

func (r *Sessions) ratingsForSession(ctx context.Context, id string) ([]practice.RatedCard, error) {
	rows, err := pg.Conn(ctx, r.pool).Query(ctx, `
SELECT card_id, rating, rated_at
FROM practice.session_card_ratings
WHERE session_id = $1
ORDER BY rated_at ASC
`, id)
	if err != nil {
		return nil, fmt.Errorf("practicerepo.Sessions.ratingsForSession: %w", err)
	}
	defer rows.Close()
	var out []practice.RatedCard
	for rows.Next() {
		rc, err := scanRating(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
