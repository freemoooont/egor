// Package practice wires the practice application use cases to HTTP.
package practice

import (
	"net/http"

	apppractice "github.com/micocards/api/internal/application/practice"
	dtopractice "github.com/micocards/api/internal/interfaces/http/dto/practice"
	"github.com/micocards/api/internal/interfaces/http/middleware"
)

// Deps groups the use cases.
type Deps struct {
	Start    apppractice.StartSession
	Rate     apppractice.RateCard
	Finish   apppractice.FinishSession
	Results  apppractice.GetResults
	Progress apppractice.GetUserDeckProgress
}

// Handlers carries the wired Deps.
type Handlers struct{ d Deps }

// New builds the handler set.
func New(d Deps) *Handlers { return &Handlers{d: d} }

// Start handles POST /api/practice/sessions.
func (h *Handlers) Start(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	var body dtopractice.StartPracticeRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	out, err := h.d.Start.Handle(r.Context(), apppractice.StartSessionInput{
		UserID: uid, DeckID: body.DeckID, Mode: body.Mode,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, dtopractice.PracticeSession{
		ID: out.SessionID, DeckID: out.DeckID, UserID: uid,
		Mode: out.Mode, Status: "in_progress",
		CardIDs: out.CardIDs, StartedAt: out.StartedAt,
	})
}

// Rate handles POST /api/practice/sessions/{sessionID}/ratings.
func (h *Handlers) Rate(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	sid := r.PathValue("sessionID")
	var body dtopractice.RateCardRequest
	if err := middleware.DecodeStrictJSON(r, &body); err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	out, err := h.d.Rate.Handle(r.Context(), apppractice.RateCardInput{
		UserID: uid, SessionID: sid, CardID: body.CardID, Rating: body.Rating,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, dtopractice.RatedCard{
		SessionID: out.SessionID, CardID: out.CardID,
		Rating: out.Rating, RatedAt: out.RatedAt,
	})
}

// Finish handles POST /api/practice/sessions/{sessionID}/finish.
func (h *Handlers) Finish(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	sid := r.PathValue("sessionID")
	out, err := h.d.Finish.Handle(r.Context(), apppractice.FinishSessionInput{
		UserID: uid, SessionID: sid,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	// Finish returns the summary; clients fetch the full results separately.
	middleware.WriteJSON(w, http.StatusOK, dtopractice.PracticeResults{
		SessionID:          out.SessionID,
		Mode:               out.Mode,
		CountDontKnow:      out.CountDontKnow,
		CountStillLearning: out.CountStillLearning,
		CountKnowKnow:      out.CountKnowKnow,
		CompletedAt:        out.CompletedAt,
		RatedCards:         []dtopractice.RatedCard{},
	})
}

// Results handles GET /api/practice/sessions/{sessionID}/results.
func (h *Handlers) Results(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	sid := r.PathValue("sessionID")
	out, err := h.d.Results.Handle(r.Context(), apppractice.GetResultsInput{
		UserID: uid, SessionID: sid,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	rated := make([]dtopractice.RatedCard, 0, len(out.RatedCards))
	for _, rc := range out.RatedCards {
		rated = append(rated, dtopractice.RatedCard{
			SessionID: out.SessionID, CardID: rc.CardID, Rating: rc.Rating,
		})
	}
	middleware.WriteJSON(w, http.StatusOK, dtopractice.PracticeResults{
		SessionID:          out.SessionID,
		DeckID:             out.DeckID,
		Mode:               out.Mode,
		CountDontKnow:      out.CountDontKnow,
		CountStillLearning: out.CountStillLearning,
		CountKnowKnow:      out.CountKnowKnow,
		RatedCards:         rated,
		CompletedAt:        out.CompletedAt,
	})
}

// Progress handles GET /api/decks/{deckID}/progress.
func (h *Handlers) Progress(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())
	deckID := r.PathValue("deckID")
	out, err := h.d.Progress.Handle(r.Context(), apppractice.GetUserDeckProgressInput{
		UserID: uid, DeckID: deckID,
	})
	if err != nil {
		middleware.WriteError(w, r, err)
		return
	}
	cards := make([]dtopractice.CardProgress, 0, len(out.CardProgress))
	for _, p := range out.CardProgress {
		cards = append(cards, dtopractice.CardProgress{
			CardID: p.CardID, Rating: p.Rating, LastRatedAt: p.LastRatedAt,
		})
	}
	middleware.WriteJSON(w, http.StatusOK, dtopractice.UserDeckProgress{
		DeckID: out.DeckID, CardProgress: cards,
	})
}
