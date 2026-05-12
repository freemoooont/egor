package decks

import "time"

// Event marker.
type Event interface {
	Name() string
}

// DeckCreated is emitted on CreateDeck.
type DeckCreated struct {
	DeckID    string
	OwnerID   string
	Title     string
	CardCount int
	CreatedAt time.Time
}

// Name returns the wire name.
func (DeckCreated) Name() string { return "decks.DeckCreated" }

// DeckRenamed is emitted on RenameDeck.
type DeckRenamed struct {
	DeckID    string
	OwnerID   string
	OldTitle  string
	NewTitle  string
	RenamedAt time.Time
}

// Name returns the wire name.
func (DeckRenamed) Name() string { return "decks.DeckRenamed" }

// DeckDeleted is emitted on DeleteDeck.
type DeckDeleted struct {
	DeckID    string
	OwnerID   string
	DeletedAt time.Time
}

// Name returns the wire name.
func (DeckDeleted) Name() string { return "decks.DeckDeleted" }

// CardAdded is emitted on AddCard.
type CardAdded struct {
	DeckID     string
	CardID     string
	Term       string
	Definition string
	Ordinal    int
	AddedAt    time.Time
}

// Name returns the wire name.
func (CardAdded) Name() string { return "decks.CardAdded" }

// CardEdited is emitted on EditCard.
type CardEdited struct {
	DeckID        string
	CardID        string
	OldTerm       string
	NewTerm       string
	OldDefinition string
	NewDefinition string
	EditedAt      time.Time
}

// Name returns the wire name.
func (CardEdited) Name() string { return "decks.CardEdited" }

// CardRemoved is emitted on RemoveCard.
type CardRemoved struct {
	DeckID    string
	CardID    string
	RemovedAt time.Time
}

// Name returns the wire name.
func (CardRemoved) Name() string { return "decks.CardRemoved" }

// CardsReordered is emitted on ReorderCards.
type CardsReordered struct {
	DeckID      string
	OrderedIDs  []string
	ReorderedAt time.Time
}

// Name returns the wire name.
func (CardsReordered) Name() string { return "decks.CardsReordered" }

// DeckGenerationRequested is emitted by GenerateDeckWithAI on entry.
type DeckGenerationRequested struct {
	RequestID   string
	OwnerID     string
	Prompt      string
	RequestedAt time.Time
}

// Name returns the wire name.
func (DeckGenerationRequested) Name() string { return "decks.DeckGenerationRequested" }

// DeckGenerationCompleted is emitted by GenerateDeckWithAI on completion.
type DeckGenerationCompleted struct {
	RequestID   string
	OwnerID     string
	Status      string
	Draft       AIDeckDraft
	CompletedAt time.Time
}

// Name returns the wire name.
func (DeckGenerationCompleted) Name() string { return "decks.DeckGenerationCompleted" }
