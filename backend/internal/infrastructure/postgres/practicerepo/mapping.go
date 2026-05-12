package practicerepo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/micocards/api/internal/domain/practice"
)

type rowScanner interface {
	Scan(dest ...any) error
}

type rawSession struct {
	id, userID, deckID string
	mode               practice.SessionMode
	status             practice.SessionStatus
	cardIDs            []string
	startedAt          time.Time
	completedAt        *time.Time
	abandonedAt        *time.Time
}

func scanSessionRow(s rowScanner) (rawSession, error) {
	var (
		id, userID, deckID, mode, status string
		cardIDsJSON                      string
		startedAt                        time.Time
		completedAt, abandonedAt         *time.Time
	)
	if err := s.Scan(&id, &userID, &deckID, &mode, &status, &cardIDsJSON, &startedAt, &completedAt, &abandonedAt); err != nil {
		return rawSession{}, err
	}
	var ids []string
	if len(cardIDsJSON) > 0 {
		if err := json.Unmarshal([]byte(cardIDsJSON), &ids); err != nil {
			return rawSession{}, fmt.Errorf("scanSessionRow card_ids: %w", err)
		}
	}
	out := rawSession{
		id: id, userID: userID, deckID: deckID,
		mode: practice.SessionMode(mode), status: practice.SessionStatus(status),
		cardIDs: ids, startedAt: startedAt,
	}
	if completedAt != nil {
		t := completedAt.UTC()
		out.completedAt = &t
	}
	if abandonedAt != nil {
		t := abandonedAt.UTC()
		out.abandonedAt = &t
	}
	return out, nil
}

func scanRating(s rowScanner) (practice.RatedCard, error) {
	var (
		cardID  string
		rating  int16
		ratedAt time.Time
	)
	if err := s.Scan(&cardID, &rating, &ratedAt); err != nil {
		return practice.RatedCard{}, err
	}
	return practice.RatedCard{
		CardID:  cardID,
		Rating:  practice.CardRating(rating),
		RatedAt: ratedAt.UTC(),
	}, nil
}
