package practice

import "time"

// CardProgress is a value object inside UserDeckProgress.
type CardProgress struct {
	CardID      string
	Rating      CardRating
	LastRatedAt time.Time
}

// UserDeckProgress is the read-model aggregate for one (user_id, deck_id) pair.
type UserDeckProgress struct {
	UserID    string
	DeckID    string
	Cards     []CardProgress
	UpdatedAt time.Time
}

// ApplySessionCompleted folds a completed session's rated cards into the read
// model. Invariants 1, 2, 3.
func (p *UserDeckProgress) ApplySessionCompleted(mode SessionMode, rated []RatedCard, at time.Time) {
	if mode != ModeTracked { // invariant 3
		return
	}
	byID := make(map[string]CardProgress, len(p.Cards))
	for _, c := range p.Cards {
		byID[c.CardID] = c
	}
	for _, r := range rated {
		byID[r.CardID] = CardProgress{
			CardID:      r.CardID,
			Rating:      r.Rating,
			LastRatedAt: r.RatedAt,
		}
	}
	out := make([]CardProgress, 0, len(byID))
	for _, c := range byID {
		out = append(out, c)
	}
	p.Cards = out
	p.UpdatedAt = at.UTC()
}

// DropCard removes a card from the read model. Invariant 4.
func (p *UserDeckProgress) DropCard(cardID string) {
	out := p.Cards[:0]
	for _, c := range p.Cards {
		if c.CardID != cardID {
			out = append(out, c)
		}
	}
	p.Cards = out
}

// IsZero reports whether the read model has any data.
func (p *UserDeckProgress) IsZero() bool { return len(p.Cards) == 0 }
