package decks_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appdecks "github.com/micocards/api/internal/application/decks"
	domdecks "github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
	httpdecks "github.com/micocards/api/internal/interfaces/http/decks"
	"github.com/micocards/api/internal/interfaces/http/middleware"
)

// fakes

type fakeDecksRepo struct {
	mu  sync.Mutex
	rec map[string]*domdecks.Deck
}

func newFakeDecksRepo() *fakeDecksRepo { return &fakeDecksRepo{rec: map[string]*domdecks.Deck{}} }

func (f *fakeDecksRepo) Save(_ context.Context, d *domdecks.Deck) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec[d.ID()] = d
	return nil
}

func (f *fakeDecksRepo) ByID(_ context.Context, id string) (*domdecks.Deck, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.rec[id]
	if !ok {
		return nil, domdecks.ErrDeckNotFound
	}
	return d, nil
}

func (f *fakeDecksRepo) ByOwner(_ context.Context, owner string, limit int, _ string) ([]*domdecks.Deck, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*domdecks.Deck{}
	for _, d := range f.rec {
		if d.OwnerID() == owner && !d.IsDeleted() {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, "", nil
}

func (f *fakeDecksRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rec, id)
	return nil
}

type fakeUoW struct{}

func (fakeUoW) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fakeEvents struct{}

func (fakeEvents) Publish(_ context.Context, _ ...domdecks.Event) error { return nil }

type fakeAI struct{ configured bool }

func (f fakeAI) IsConfigured() bool { return f.configured }
func (f fakeAI) Generate(_ context.Context, _ string) (domdecks.AIDeckDraft, error) {
	return domdecks.AIDeckDraft{}, domdecks.ErrAINotConfigured
}

func buildHandlers(t *testing.T, aiOn bool) (*httpdecks.Handlers, *fakeDecksRepo) {
	t.Helper()
	repo := newFakeDecksRepo()
	clk := clock.NewFixed(time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC))
	ids := idgen.NewSequential("deck")
	d := httpdecks.Deps{
		List:       appdecks.ListUserDecks{Decks: repo},
		Create:     appdecks.CreateDeck{Decks: repo, IDs: ids, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		Get:        appdecks.GetDeck{Decks: repo},
		Rename:     appdecks.RenameDeck{Decks: repo, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		Delete:     appdecks.DeleteDeck{Decks: repo, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		AddCard:    appdecks.AddCard{Decks: repo, IDs: ids, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		EditCard:   appdecks.EditCard{Decks: repo, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		RemoveCard: appdecks.RemoveCard{Decks: repo, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		Reorder:    appdecks.ReorderCards{Decks: repo, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		Generate:   appdecks.GenerateDeckWithAI{AI: fakeAI{configured: aiOn}, IDs: ids, Clock: clk, Events: fakeEvents{}},
		AIEnabled:  aiOn,
	}
	return httpdecks.New(d), repo
}

func authedReq(method, path string, body any, deckID, cardID string) *http.Request {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithUserID(req.Context(), "owner-1"))
	if deckID != "" {
		req.SetPathValue("deckID", deckID)
	}
	if cardID != "" {
		req.SetPathValue("cardID", cardID)
	}
	return req
}

func TestCreateDeck_201(t *testing.T) {
	h, _ := buildHandlers(t, false)
	body := map[string]any{
		"title": "My deck",
		"cards": []map[string]string{{"term": "t1", "definition": "d1"}},
	}
	req := authedReq("POST", "/api/decks", body, "", "")
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Create).ServeHTTP(rr, req)
	require.Equal(t, 201, rr.Code, rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, "My deck", got["title"])
}

func TestCreateDeck_422_EmptyTitle(t *testing.T) {
	h, _ := buildHandlers(t, false)
	req := authedReq("POST", "/api/decks", map[string]any{"title": ""}, "", "")
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Create).ServeHTTP(rr, req)
	require.Equal(t, 422, rr.Code)
}

func TestList_200(t *testing.T) {
	h, _ := buildHandlers(t, false)
	// Create one
	req := authedReq("POST", "/api/decks", map[string]any{"title": "L1"}, "", "")
	http.HandlerFunc(h.Create).ServeHTTP(httptest.NewRecorder(), req)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.List).ServeHTTP(rr, authedReq("GET", "/api/decks", nil, "", ""))
	require.Equal(t, 200, rr.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	decks := resp["decks"].([]any)
	require.Len(t, decks, 1)
}

func TestGet_404(t *testing.T) {
	h, _ := buildHandlers(t, false)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Get).ServeHTTP(rr, authedReq("GET", "/api/decks/x", nil, "missing", ""))
	require.Equal(t, 404, rr.Code)
}

func TestGet_200(t *testing.T) {
	h, _ := buildHandlers(t, false)
	rrCreate := httptest.NewRecorder()
	http.HandlerFunc(h.Create).ServeHTTP(rrCreate, authedReq("POST", "/api/decks", map[string]any{"title": "G1"}, "", ""))
	require.Equal(t, 201, rrCreate.Code)
	var d map[string]any
	require.NoError(t, json.Unmarshal(rrCreate.Body.Bytes(), &d))
	id := d["id"].(string)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Get).ServeHTTP(rr, authedReq("GET", "/api/decks/x", nil, id, ""))
	require.Equal(t, 200, rr.Code)
}

func TestRename_200(t *testing.T) {
	h, _ := buildHandlers(t, false)
	rrCreate := httptest.NewRecorder()
	http.HandlerFunc(h.Create).ServeHTTP(rrCreate, authedReq("POST", "/api/decks", map[string]any{"title": "old"}, "", ""))
	var d map[string]any
	require.NoError(t, json.Unmarshal(rrCreate.Body.Bytes(), &d))
	id := d["id"].(string)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Rename).ServeHTTP(rr, authedReq("PUT", "/api/decks/x", map[string]any{"title": "new"}, id, ""))
	require.Equal(t, 200, rr.Code, rr.Body.String())
}

func TestDelete_OK(t *testing.T) {
	h, _ := buildHandlers(t, false)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Delete).ServeHTTP(rr, authedReq("DELETE", "/api/decks/x", nil, "missing", ""))
	require.Equal(t, 200, rr.Code) // idempotent
}

func TestAddCard_201(t *testing.T) {
	h, _ := buildHandlers(t, false)
	rrCreate := httptest.NewRecorder()
	http.HandlerFunc(h.Create).ServeHTTP(rrCreate, authedReq("POST", "/api/decks", map[string]any{"title": "X"}, "", ""))
	var d map[string]any
	require.NoError(t, json.Unmarshal(rrCreate.Body.Bytes(), &d))
	id := d["id"].(string)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.AddCard).ServeHTTP(rr, authedReq("POST", "/api/decks/x/cards",
		map[string]any{"term": "t", "definition": "def"}, id, ""))
	require.Equal(t, 201, rr.Code, rr.Body.String())
}

func TestEditCard_404(t *testing.T) {
	h, _ := buildHandlers(t, false)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.EditCard).ServeHTTP(rr, authedReq("PUT", "/api/decks/x/cards/y",
		map[string]any{"term": "t"}, "missing", "card-1"))
	require.Equal(t, 404, rr.Code)
}

func TestRemoveCard_OK(t *testing.T) {
	h, _ := buildHandlers(t, false)
	rrCreate := httptest.NewRecorder()
	http.HandlerFunc(h.Create).ServeHTTP(rrCreate, authedReq("POST", "/api/decks",
		map[string]any{"title": "rc", "cards": []map[string]string{{"term": "a", "definition": "b"}}}, "", ""))
	var d map[string]any
	require.NoError(t, json.Unmarshal(rrCreate.Body.Bytes(), &d))
	id := d["id"].(string)
	cards := d["cards"].([]any)
	cardID := cards[0].(map[string]any)["id"].(string)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.RemoveCard).ServeHTTP(rr, authedReq("DELETE", "/api/decks/x/cards/y", nil, id, cardID))
	require.Equal(t, 200, rr.Code)
}

func TestReorder_200(t *testing.T) {
	h, _ := buildHandlers(t, false)
	body := map[string]any{
		"title": "ro",
		"cards": []map[string]string{
			{"term": "1", "definition": "1"}, {"term": "2", "definition": "2"},
		},
	}
	rrCreate := httptest.NewRecorder()
	http.HandlerFunc(h.Create).ServeHTTP(rrCreate, authedReq("POST", "/api/decks", body, "", ""))
	var d map[string]any
	require.NoError(t, json.Unmarshal(rrCreate.Body.Bytes(), &d))
	id := d["id"].(string)
	cards := d["cards"].([]any)
	id1 := cards[0].(map[string]any)["id"].(string)
	id2 := cards[1].(map[string]any)["id"].(string)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Reorder).ServeHTTP(rr, authedReq("POST", "/api/decks/x/reorder",
		map[string]any{"orderedIds": []string{id2, id1}}, id, ""))
	require.Equal(t, 200, rr.Code, rr.Body.String())
}

func TestGenerate_501_WhenAIOff(t *testing.T) {
	h, _ := buildHandlers(t, false)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Generate).ServeHTTP(rr, authedReq("POST", "/api/decks/generate",
		map[string]any{"prompt": "hello"}, "", ""))
	require.Equal(t, 501, rr.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, "ai_not_configured", body["error"])
}

func TestGenerate_502_WhenAIOnButProviderErrors(t *testing.T) {
	// AIEnabled=true but the stub still errors → use case maps to ErrAIUpstream → 502.
	h, _ := buildHandlers(t, true)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Generate).ServeHTTP(rr, authedReq("POST", "/api/decks/generate",
		map[string]any{"prompt": "hello"}, "", ""))
	require.Equal(t, 502, rr.Code)
}
