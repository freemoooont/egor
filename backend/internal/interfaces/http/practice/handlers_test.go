package practice_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppractice "github.com/micocards/api/internal/application/practice"
	domdecks "github.com/micocards/api/internal/domain/decks"
	dompractice "github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
	"github.com/micocards/api/internal/interfaces/http/middleware"
	httppractice "github.com/micocards/api/internal/interfaces/http/practice"
)

type fakeSessions struct {
	mu  sync.Mutex
	rec map[string]*dompractice.Session
}

func newFakeSessions() *fakeSessions { return &fakeSessions{rec: map[string]*dompractice.Session{}} }

func (f *fakeSessions) Save(_ context.Context, s *dompractice.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec[s.ID()] = s
	return nil
}

func (f *fakeSessions) ByID(_ context.Context, id string) (*dompractice.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.rec[id]
	if !ok {
		return nil, dompractice.ErrSessionNotFound
	}
	return s, nil
}

func (f *fakeSessions) LatestCompletedFor(_ context.Context, _, _ string) (*dompractice.Session, error) {
	return nil, dompractice.ErrSessionNotFound
}

type fakeProgress struct {
	mu sync.Mutex
	by map[string]*dompractice.UserDeckProgress
}

func newFakeProgress() *fakeProgress {
	return &fakeProgress{by: map[string]*dompractice.UserDeckProgress{}}
}

func (f *fakeProgress) ByUserAndDeck(_ context.Context, u, d string) (*dompractice.UserDeckProgress, error) {
	return f.by[u+"|"+d], nil
}
func (f *fakeProgress) Save(_ context.Context, p *dompractice.UserDeckProgress) error {
	f.by[p.UserID+"|"+p.DeckID] = p
	return nil
}
func (f *fakeProgress) DeleteByDeck(_ context.Context, _ string) error { return nil }

type fakeSnap struct {
	owner string
	cards []string
	miss  bool
}

func (f fakeSnap) OwnerAndCards(_ context.Context, _ string) (string, []string, error) {
	if f.miss {
		return "", nil, domdecks.ErrDeckNotFound
	}
	return f.owner, f.cards, nil
}

type fakeUoW struct{}

func (fakeUoW) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fakeEvents struct{}

func (fakeEvents) Publish(_ context.Context, _ ...dompractice.Event) error { return nil }

func buildHandlers(t *testing.T, snap fakeSnap) (*httppractice.Handlers, *fakeSessions) {
	t.Helper()
	sessions := newFakeSessions()
	clk := clock.NewFixed(time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC))
	ids := idgen.NewSequential("session")
	d := httppractice.Deps{
		Start:    apppractice.StartSession{Sessions: sessions, Snapshots: snap, IDs: ids, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		Rate:     apppractice.RateCard{Sessions: sessions, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		Finish:   apppractice.FinishSession{Sessions: sessions, Clock: clk, UoW: fakeUoW{}, Events: fakeEvents{}},
		Results:  apppractice.GetResults{Sessions: sessions},
		Progress: apppractice.GetUserDeckProgress{Progress: newFakeProgress()},
	}
	return httppractice.New(d), sessions
}

func authedReq(method, path string, body any, sessionID, deckID string) *http.Request {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithUserID(req.Context(), "u-from-token"))
	if sessionID != "" {
		req.SetPathValue("sessionID", sessionID)
	}
	if deckID != "" {
		req.SetPathValue("deckID", deckID)
	}
	return req
}

func TestStart_201(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "u-from-token", cards: []string{"c1", "c2"}})
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Start).ServeHTTP(rr, authedReq("POST", "/api/practice/sessions",
		map[string]any{"deckId": "d1", "mode": "tracked"}, "", ""))
	require.Equal(t, 201, rr.Code, rr.Body.String())
	var b map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &b))
	require.Equal(t, "in_progress", b["status"])
}

func TestStart_403_NotOwner(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "someone-else", cards: []string{"c1"}})
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Start).ServeHTTP(rr, authedReq("POST", "/api/practice/sessions",
		map[string]any{"deckId": "d1", "mode": "tracked"}, "", ""))
	require.Equal(t, 403, rr.Code)
}

func TestStart_404_DeckMissing(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{miss: true})
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Start).ServeHTTP(rr, authedReq("POST", "/api/practice/sessions",
		map[string]any{"deckId": "d1", "mode": "tracked"}, "", ""))
	require.Equal(t, 404, rr.Code)
}

func TestStart_422_DeckEmpty(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "u-from-token", cards: []string{}})
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Start).ServeHTTP(rr, authedReq("POST", "/api/practice/sessions",
		map[string]any{"deckId": "d1", "mode": "tracked"}, "", ""))
	require.Equal(t, 422, rr.Code)
}

func TestStart_422_BadMode(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "u-from-token", cards: []string{"c1"}})
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Start).ServeHTTP(rr, authedReq("POST", "/api/practice/sessions",
		map[string]any{"deckId": "d1", "mode": "wrong"}, "", ""))
	require.Equal(t, 422, rr.Code)
}

func TestRate_OK_AndFinishAndResults(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "u-from-token", cards: []string{"c1", "c2"}})
	// start
	rrStart := httptest.NewRecorder()
	http.HandlerFunc(h.Start).ServeHTTP(rrStart, authedReq("POST", "/api/practice/sessions",
		map[string]any{"deckId": "d1", "mode": "tracked"}, "", ""))
	require.Equal(t, 201, rrStart.Code)
	var s map[string]any
	require.NoError(t, json.Unmarshal(rrStart.Body.Bytes(), &s))
	sid := s["id"].(string)

	// rate twice
	for _, cardID := range []string{"c1", "c2"} {
		rr := httptest.NewRecorder()
		http.HandlerFunc(h.Rate).ServeHTTP(rr, authedReq("POST", "/api/practice/sessions/x/ratings",
			map[string]any{"cardId": cardID, "rating": 2}, sid, ""))
		require.Equal(t, 200, rr.Code, rr.Body.String())
	}

	// finish
	rrFin := httptest.NewRecorder()
	http.HandlerFunc(h.Finish).ServeHTTP(rrFin, authedReq("POST", "/api/practice/sessions/x/finish", nil, sid, ""))
	require.Equal(t, 200, rrFin.Code)
	var fin map[string]any
	require.NoError(t, json.Unmarshal(rrFin.Body.Bytes(), &fin))
	require.EqualValues(t, 2, fin["countKnowKnow"])

	// results
	rrRes := httptest.NewRecorder()
	http.HandlerFunc(h.Results).ServeHTTP(rrRes, authedReq("GET", "/api/practice/sessions/x/results", nil, sid, ""))
	require.Equal(t, 200, rrRes.Code, rrRes.Body.String())
}

func TestRate_404_MissingSession(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "u-from-token", cards: []string{"c1"}})
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Rate).ServeHTTP(rr, authedReq("POST", "/api/practice/sessions/x/ratings",
		map[string]any{"cardId": "c1", "rating": 0}, "no-such-session", ""))
	require.Equal(t, 404, rr.Code)
}

func TestRate_422_BadRating(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "u-from-token", cards: []string{"c1"}})
	rrStart := httptest.NewRecorder()
	http.HandlerFunc(h.Start).ServeHTTP(rrStart, authedReq("POST", "/api/practice/sessions",
		map[string]any{"deckId": "d1", "mode": "tracked"}, "", ""))
	var s map[string]any
	require.NoError(t, json.Unmarshal(rrStart.Body.Bytes(), &s))
	sid := s["id"].(string)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Rate).ServeHTTP(rr, authedReq("POST", "/api/practice/sessions/x/ratings",
		map[string]any{"cardId": "c1", "rating": 9}, sid, ""))
	require.Equal(t, 422, rr.Code)
}

func TestResults_409_NotCompleted(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "u-from-token", cards: []string{"c1"}})
	rrStart := httptest.NewRecorder()
	http.HandlerFunc(h.Start).ServeHTTP(rrStart, authedReq("POST", "/api/practice/sessions",
		map[string]any{"deckId": "d1", "mode": "tracked"}, "", ""))
	var s map[string]any
	require.NoError(t, json.Unmarshal(rrStart.Body.Bytes(), &s))
	sid := s["id"].(string)
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Results).ServeHTTP(rr, authedReq("GET", "/api/practice/sessions/x/results", nil, sid, ""))
	require.Equal(t, 409, rr.Code)
}

func TestProgress_200(t *testing.T) {
	h, _ := buildHandlers(t, fakeSnap{owner: "u-from-token", cards: []string{"c1"}})
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Progress).ServeHTTP(rr, authedReq("GET", "/api/decks/x/progress", nil, "", "deck-1"))
	require.Equal(t, 200, rr.Code, rr.Body.String())
}
