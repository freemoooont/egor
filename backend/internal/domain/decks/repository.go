package decks

import "context"

// Decks is the persistence port for the Deck aggregate.
type Decks interface {
	Save(ctx context.Context, d *Deck) error
	ByID(ctx context.Context, deckID string) (*Deck, error)
	ByOwner(ctx context.Context, ownerID string, limit int, cursor string) ([]*Deck, string, error)
	Delete(ctx context.Context, deckID string) error
}

// AIDeckDraft is the v2 output shape; v1 returns it empty under
// status="not_implemented" (see GenerateDeckWithAI use case).
type AIDeckDraft struct {
	Title string
	Cards []AIDraftCard
}

// AIDraftCard is one card in the AI draft.
type AIDraftCard struct {
	Term       string
	Definition string
}
