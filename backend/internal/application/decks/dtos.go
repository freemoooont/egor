package decks

import (
	"time"

	"github.com/micocards/api/internal/domain/decks"
)

// CardView is the application-layer projection of a Card.
type CardView struct {
	ID         string
	Term       string
	Definition string
	Ordinal    int
}

func cardViewOf(c decks.Card) CardView {
	return CardView{ID: c.ID(), Term: c.Term().String(), Definition: c.Definition().String(), Ordinal: c.Ordinal()}
}

// DeckView is the application-layer projection of a Deck.
type DeckView struct {
	DeckID    string
	OwnerID   string
	Title     string
	Cards     []CardView
	CreatedAt time.Time
}

func deckViewOf(d *decks.Deck) DeckView {
	cards := d.Cards()
	out := DeckView{
		DeckID: d.ID(), OwnerID: d.OwnerID(), Title: d.Title().String(),
		CreatedAt: d.CreatedAt(),
		Cards:     make([]CardView, 0, len(cards)),
	}
	for _, c := range cards {
		out.Cards = append(out.Cards, cardViewOf(c))
	}
	return out
}

// DeckSummary is the trimmed shape used by ListUserDecks.
type DeckSummary struct {
	DeckID    string
	Title     string
	CardCount int
	CreatedAt time.Time
}

// CreateDeckInput is the request payload.
type CreateDeckInput struct {
	OwnerID string
	Title   string
	Cards   []DraftCard
}

// DraftCard is one card supplied at deck creation.
type DraftCard struct {
	Term       string
	Definition string
}

// CreateDeckOutput is the response payload.
type CreateDeckOutput struct {
	Deck DeckView
}

// RenameDeckInput / Output.
type RenameDeckInput struct {
	OwnerID string
	DeckID  string
	Title   string
}

// RenameDeckOutput is the response payload.
type RenameDeckOutput struct {
	DeckID    string
	Title     string
	RenamedAt time.Time
}

// DeleteDeckInput / Output.
type DeleteDeckInput struct {
	OwnerID string
	DeckID  string
}

// DeleteDeckOutput is the response payload.
type DeleteDeckOutput struct {
	OK bool
}

// AddCardInput / Output.
type AddCardInput struct {
	OwnerID    string
	DeckID     string
	Term       string
	Definition string
}

// AddCardOutput is the response payload.
type AddCardOutput struct {
	Card CardView
}

// EditCardInput / Output.
type EditCardInput struct {
	OwnerID    string
	DeckID     string
	CardID     string
	Term       *string
	Definition *string
}

// EditCardOutput is the response payload.
type EditCardOutput struct {
	Card CardView
}

// RemoveCardInput / Output.
type RemoveCardInput struct {
	OwnerID string
	DeckID  string
	CardID  string
}

// RemoveCardOutput is the response payload.
type RemoveCardOutput struct {
	OK bool
}

// ReorderCardsInput / Output.
type ReorderCardsInput struct {
	OwnerID    string
	DeckID     string
	OrderedIDs []string
}

// ReorderCardsOutput is the response payload.
type ReorderCardsOutput struct {
	Cards []CardView
}

// GetDeckInput / Output.
type GetDeckInput struct {
	OwnerID string
	DeckID  string
}

// GetDeckOutput is the response payload.
type GetDeckOutput struct {
	Deck DeckView
}

// ListUserDecksInput / Output.
type ListUserDecksInput struct {
	OwnerID string
	Limit   int
	Cursor  string
}

// ListUserDecksOutput is the response payload.
type ListUserDecksOutput struct {
	Decks      []DeckSummary
	NextCursor string
}

// GenerateDeckWithAIInput / Output.
type GenerateDeckWithAIInput struct {
	OwnerID   string
	Prompt    string
	RequestID string
}

// AIDeckDraftView is the application-layer projection of decks.AIDeckDraft.
type AIDeckDraftView struct {
	Title string
	Cards []DraftCard
}

// GenerateDeckWithAIOutput is the response payload.
type GenerateDeckWithAIOutput struct {
	Status string
	Draft  AIDeckDraftView
}
