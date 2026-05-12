// Package decks contains the HTTP DTOs for the decks context.
package decks

import "time"

// Card mirrors the OpenAPI Card schema.
type Card struct {
	ID         string `json:"id"`
	Term       string `json:"term"`
	Definition string `json:"definition"`
	Ordinal    int    `json:"ordinal"`
}

// CardDraft is one card on POST/PUT.
type CardDraft struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}

// Deck mirrors the OpenAPI Deck schema.
type Deck struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"ownerId"`
	Title     string    `json:"title"`
	Cards     []Card    `json:"cards"`
	CreatedAt time.Time `json:"createdAt"`
}

// DeckSummary is the trimmed shape used by ListUserDecks.
type DeckSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CardCount int       `json:"cardCount"`
	CreatedAt time.Time `json:"createdAt"`
}

// DeckList is the paginated list response.
type DeckList struct {
	Decks      []DeckSummary `json:"decks"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

// CardList is the response for reorder.
type CardList struct {
	Cards []Card `json:"cards"`
}

// CreateDeckRequest is the body for POST /api/decks.
type CreateDeckRequest struct {
	Title string      `json:"title"`
	Cards []CardDraft `json:"cards,omitempty"`
}

// RenameDeckRequest is the body for PATCH/PUT /api/decks/{id}.
type RenameDeckRequest struct {
	Title string `json:"title"`
}

// AddCardRequest is the body for POST /api/decks/{id}/cards.
type AddCardRequest struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}

// EditCardRequest is the body for PATCH/PUT /api/decks/{id}/cards/{cardId}.
type EditCardRequest struct {
	Term       *string `json:"term,omitempty"`
	Definition *string `json:"definition,omitempty"`
}

// ReorderCardsRequest is the body for POST/PUT /api/decks/{id}/reorder
// (a.k.a. /api/decks/{id}/cards/order).
type ReorderCardsRequest struct {
	OrderedIDs []string `json:"orderedIds"`
}

// GenerateDeckRequest is the body for POST /api/decks/generate.
type GenerateDeckRequest struct {
	Prompt string `json:"prompt"`
}

// AIDeckDraftResponse is the response for POST /api/decks/generate.
type AIDeckDraftResponse struct {
	Draft AIDeckDraft `json:"draft"`
}

// AIDeckDraft is the title+cards shape returned by the AI generator.
type AIDeckDraft struct {
	Title string      `json:"title"`
	Cards []CardDraft `json:"cards"`
}
