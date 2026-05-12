// Package decks wires the decks application use cases to HTTP.
package decks

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	appdecks "github.com/micocards/api/internal/application/decks"
	domdecks "github.com/micocards/api/internal/domain/decks"
	dtodecks "github.com/micocards/api/internal/interfaces/http/dto/decks"
	"github.com/micocards/api/internal/interfaces/http/middleware"
)

// Deps groups the use cases.
type Deps struct {
	List       appdecks.ListUserDecks
	Create     appdecks.CreateDeck
	Get        appdecks.GetDeck
	Rename     appdecks.RenameDeck
	Delete     appdecks.DeleteDeck
	AddCard    appdecks.AddCard
	EditCard   appdecks.EditCard
	RemoveCard appdecks.RemoveCard
	Reorder    appdecks.ReorderCards
	Generate   appdecks.GenerateDeckWithAI
	AIEnabled  bool
}

// Handlers carries the wired Deps.
type Handlers struct{ d Deps }

// New builds the handler set.
func New(d Deps) *Handlers { return &Handlers{d: d} }

// List handles GET /api/decks.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	cursor := r.URL.Query().Get("cursor")
	out, err := h.d.List.Handle(r.Context(), appdecks.ListUserDecksInput{
		OwnerID: uid, Limit: limit, Cursor: cursor,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	resp := dtodecks.DeckList{NextCursor: out.NextCursor, Decks: make([]dtodecks.DeckSummary, 0, len(out.Decks))}
	for _, d := range out.Decks {
		resp.Decks = append(resp.Decks, dtodecks.DeckSummary{
			ID: d.DeckID, Title: d.Title, CardCount: d.CardCount, CreatedAt: d.CreatedAt,
		})
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// Create handles POST /api/decks.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var body dtodecks.CreateDeckRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		middleware.WriteError(w, r, domdecks.ErrInvalidDeckTitle)
		return
	}
	in := appdecks.CreateDeckInput{OwnerID: uid, Title: body.Title}
	for _, c := range body.Cards {
		in.Cards = append(in.Cards, appdecks.DraftCard{Term: c.Term, Definition: c.Definition})
	}
	out, err := h.d.Create.Handle(r.Context(), in)
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, deckDTO(out.Deck))
}

// Get handles GET /api/decks/{deckID}.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	out, err := h.d.Get.Handle(r.Context(), appdecks.GetDeckInput{OwnerID: uid, DeckID: deckID})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, deckDTO(out.Deck))
}

// Rename handles PUT/PATCH /api/decks/{deckID}.
func (h *Handlers) Rename(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	var body dtodecks.RenameDeckRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	out, err := h.d.Rename.Handle(r.Context(), appdecks.RenameDeckInput{
		OwnerID: uid, DeckID: deckID, Title: body.Title,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	// Reload the deck to return the full DTO. (The use case only returns
	// {DeckID, Title, RenamedAt} — but the HTTP contract sends the full Deck.)
	deckOut, err := h.d.Get.Handle(r.Context(), appdecks.GetDeckInput{OwnerID: uid, DeckID: deckID})
	if err != nil {
		// Fallback: return the trimmed shape if the read fails.
		middleware.WriteJSON(w, http.StatusOK, map[string]any{
			"id":         out.DeckID,
			"title":      out.Title,
			"renamedAt":  out.RenamedAt,
		})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, deckDTO(deckOut.Deck))
}

// Delete handles DELETE /api/decks/{deckID}.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	out, err := h.d.Delete.Handle(r.Context(), appdecks.DeleteDeckInput{OwnerID: uid, DeckID: deckID})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"ok": out.OK})
}

// AddCard handles POST /api/decks/{deckID}/cards.
func (h *Handlers) AddCard(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	var body dtodecks.AddCardRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	out, err := h.d.AddCard.Handle(r.Context(), appdecks.AddCardInput{
		OwnerID: uid, DeckID: deckID, Term: body.Term, Definition: body.Definition,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, cardDTO(out.Card))
}

// EditCard handles PUT/PATCH /api/decks/{deckID}/cards/{cardID}.
func (h *Handlers) EditCard(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	cardID := r.PathValue("cardID")
	var body dtodecks.EditCardRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	in := appdecks.EditCardInput{OwnerID: uid, DeckID: deckID, CardID: cardID}
	if body.Term != nil {
		v := *body.Term
		in.Term = &v
	}
	if body.Definition != nil {
		v := *body.Definition
		in.Definition = &v
	}
	out, err := h.d.EditCard.Handle(r.Context(), in)
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, cardDTO(out.Card))
}

// RemoveCard handles DELETE /api/decks/{deckID}/cards/{cardID}.
func (h *Handlers) RemoveCard(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	cardID := r.PathValue("cardID")
	out, err := h.d.RemoveCard.Handle(r.Context(), appdecks.RemoveCardInput{
		OwnerID: uid, DeckID: deckID, CardID: cardID,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{"ok": out.OK})
}

// Reorder handles POST /api/decks/{deckID}/reorder (and PUT
// /api/decks/{deckID}/cards/order from the OpenAPI doc).
func (h *Handlers) Reorder(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	var body dtodecks.ReorderCardsRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	out, err := h.d.Reorder.Handle(r.Context(), appdecks.ReorderCardsInput{
		OwnerID: uid, DeckID: deckID, OrderedIDs: body.OrderedIDs,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	cards := make([]dtodecks.Card, 0, len(out.Cards))
	for _, c := range out.Cards {
		cards = append(cards, cardDTO(c))
	}
	middleware.WriteJSON(w, http.StatusOK, dtodecks.CardList{Cards: cards})
}

// Generate handles POST /api/decks/generate.
//
// AI mock-only rule: if AI_API_KEY is empty (AIEnabled=false), return 501
// `{error:"ai_not_configured"}` directly without calling the use case. When
// AI_API_KEY is set, we still call the use case and let its provider return
// ErrAINotConfigured (the v1 OpenAI stub does not make a real LLM call).
func (h *Handlers) Generate(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var body dtodecks.GenerateDeckRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	if !h.d.AIEnabled {
		middleware.WriteError(w, r, domdecks.ErrAINotConfigured)
		return
	}
	out, err := h.d.Generate.Handle(r.Context(), appdecks.GenerateDeckWithAIInput{
		OwnerID: uid, Prompt: body.Prompt,
		RequestID: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		// Map ErrAINotConfigured → 501 by sentinel (already in errorMapper).
		if errors.Is(err, domdecks.ErrAINotConfigured) {
			middleware.WriteError(w, r, err)
			return
		}
		middleware.WriteError(w, r, err)
		return
	}
	resp := dtodecks.AIDeckDraftResponse{Draft: dtodecks.AIDeckDraft{Title: out.Draft.Title}}
	for _, c := range out.Draft.Cards {
		resp.Draft.Cards = append(resp.Draft.Cards, dtodecks.CardDraft{Term: c.Term, Definition: c.Definition})
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func deckDTO(d appdecks.DeckView) dtodecks.Deck {
	cards := make([]dtodecks.Card, 0, len(d.Cards))
	for _, c := range d.Cards {
		cards = append(cards, cardDTO(c))
	}
	return dtodecks.Deck{
		ID: d.DeckID, OwnerID: d.OwnerID, Title: d.Title, Cards: cards, CreatedAt: d.CreatedAt,
	}
}

func cardDTO(c appdecks.CardView) dtodecks.Card {
	return dtodecks.Card{ID: c.ID, Term: c.Term, Definition: c.Definition, Ordinal: c.Ordinal}
}
