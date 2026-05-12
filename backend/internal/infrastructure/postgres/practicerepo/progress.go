package practicerepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/micocards/api/internal/domain/practice"
	pg "github.com/micocards/api/internal/infrastructure/postgres"
)

// UserDeckProgresses is the practice.UserDeckProgresses repository.
type UserDeckProgresses struct {
	pool *pgxpool.Pool
}

// NewUserDeckProgresses builds the repo.
func NewUserDeckProgresses(pool *pgxpool.Pool) *UserDeckProgresses {
	return &UserDeckProgresses{pool: pool}
}

type cardProgressDTO struct {
	CardID      string    `json:"card_id"`
	Rating      int16     `json:"rating"`
	LastRatedAt time.Time `json:"last_rated_at"`
}

// Save upserts the per-(user_id, deck_id) progress row.
func (r *UserDeckProgresses) Save(ctx context.Context, p *practice.UserDeckProgress) error {
	cards := p.Cards
	dtos := make([]cardProgressDTO, 0, len(cards))
	var know, learning, dontKnow int
	for _, c := range cards {
		dtos = append(dtos, cardProgressDTO{
			CardID:      c.CardID,
			Rating:      int16(c.Rating),
			LastRatedAt: c.LastRatedAt.UTC(),
		})
		switch c.Rating {
		case practice.RatingKnowKnow:
			know++
		case practice.RatingStillLearning:
			learning++
		case practice.RatingDontKnow:
			dontKnow++
		}
	}
	payload, err := json.Marshal(dtos)
	if err != nil {
		return fmt.Errorf("practicerepo.UserDeckProgresses.Save marshal: %w", err)
	}
	if _, err := pg.Conn(ctx, r.pool).Exec(ctx, `
INSERT INTO practice.user_deck_progress (
    user_id, deck_id, cards, know_count, learning_count, dont_know_count, last_practiced_at, updated_at
) VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7, now())
ON CONFLICT (user_id, deck_id) DO UPDATE SET
    cards = EXCLUDED.cards,
    know_count = EXCLUDED.know_count,
    learning_count = EXCLUDED.learning_count,
    dont_know_count = EXCLUDED.dont_know_count,
    last_practiced_at = EXCLUDED.last_practiced_at,
    updated_at = now()
`,
		p.UserID, p.DeckID, string(payload),
		know, learning, dontKnow, p.UpdatedAt,
	); err != nil {
		return fmt.Errorf("practicerepo.UserDeckProgresses.Save: %w", err)
	}
	return nil
}

// ByUserAndDeck loads the progress row, returning a zero-valued UserDeckProgress
// (no error) when the row is missing — matches the read-model contract.
func (r *UserDeckProgresses) ByUserAndDeck(ctx context.Context, userID, deckID string) (*practice.UserDeckProgress, error) {
	row := pg.Conn(ctx, r.pool).QueryRow(ctx, `
SELECT cards, last_practiced_at FROM practice.user_deck_progress
WHERE user_id = $1 AND deck_id = $2
`, userID, deckID)
	var (
		raw       string
		updatedAt time.Time
	)
	if err := row.Scan(&raw, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &practice.UserDeckProgress{UserID: userID, DeckID: deckID}, nil
		}
		return nil, fmt.Errorf("practicerepo.UserDeckProgresses.ByUserAndDeck: %w", err)
	}
	var dtos []cardProgressDTO
	if len(raw) > 0 {
		if err := json.Unmarshal([]byte(raw), &dtos); err != nil {
			return nil, fmt.Errorf("practicerepo.UserDeckProgresses.ByUserAndDeck unmarshal: %w", err)
		}
	}
	cards := make([]practice.CardProgress, 0, len(dtos))
	for _, d := range dtos {
		cards = append(cards, practice.CardProgress{
			CardID:      d.CardID,
			Rating:      practice.CardRating(d.Rating),
			LastRatedAt: d.LastRatedAt.UTC(),
		})
	}
	return &practice.UserDeckProgress{
		UserID:    userID,
		DeckID:    deckID,
		Cards:     cards,
		UpdatedAt: updatedAt.UTC(),
	}, nil
}

// DeleteByDeck removes every row whose deck_id matches.
func (r *UserDeckProgresses) DeleteByDeck(ctx context.Context, deckID string) error {
	if _, err := pg.Conn(ctx, r.pool).Exec(ctx,
		`DELETE FROM practice.user_deck_progress WHERE deck_id = $1`, deckID,
	); err != nil {
		return fmt.Errorf("practicerepo.UserDeckProgresses.DeleteByDeck: %w", err)
	}
	return nil
}
