package http_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appdecks "github.com/micocards/api/internal/application/decks"
	appiam "github.com/micocards/api/internal/application/iam"
	apppractice "github.com/micocards/api/internal/application/practice"
	domdecks "github.com/micocards/api/internal/domain/decks"
	domiam "github.com/micocards/api/internal/domain/iam"
	dompractice "github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
	httproot "github.com/micocards/api/internal/interfaces/http"
	httpdecks "github.com/micocards/api/internal/interfaces/http/decks"
	httpiam "github.com/micocards/api/internal/interfaces/http/iam"
	"github.com/micocards/api/internal/interfaces/http/middleware"
	httppractice "github.com/micocards/api/internal/interfaces/http/practice"
)

// minimal in-memory fakes copied from sibling packages -----------------------

type fUsers struct {
	mu      sync.Mutex
	byID    map[string]*domiam.User
	byEmail map[string]*domiam.User
}

func newU() *fUsers {
	return &fUsers{byID: map[string]*domiam.User{}, byEmail: map[string]*domiam.User{}}
}

func (f *fUsers) Save(_ context.Context, u *domiam.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.byEmail[u.Email().String()]; ok && existing.ID() != u.ID() {
		return domiam.ErrEmailTaken
	}
	if old, ok := f.byID[u.ID()]; ok {
		delete(f.byEmail, old.Email().String())
	}
	f.byID[u.ID()] = u
	f.byEmail[u.Email().String()] = u
	return nil
}

func (f *fUsers) ByID(_ context.Context, id string) (*domiam.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domiam.ErrUserNotFound
}

func (f *fUsers) ByEmail(_ context.Context, e domiam.EmailAddress) (*domiam.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.byEmail[e.String()]; ok {
		return u, nil
	}
	return nil, domiam.ErrUserNotFound
}

func (f *fUsers) EmailExists(_ context.Context, e domiam.EmailAddress) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.byEmail[e.String()]
	return ok, nil
}

type fRefresh struct {
	mu     sync.Mutex
	byID   map[string]domiam.RefreshToken
	byHash map[string]string
}

func newR() *fRefresh {
	return &fRefresh{byID: map[string]domiam.RefreshToken{}, byHash: map[string]string{}}
}

func (f *fRefresh) Save(_ context.Context, t domiam.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[t.ID] = t
	f.byHash[t.OpaqueHash] = t.ID
	return nil
}
func (f *fRefresh) ByOpaqueHash(_ context.Context, h string) (domiam.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byHash[h]
	if !ok {
		return domiam.RefreshToken{}, domiam.ErrRefreshTokenInvalid
	}
	return f.byID[id], nil
}
func (f *fRefresh) FamilyByID(_ context.Context, id string) (domiam.RefreshTokenFamily, error) {
	return domiam.RefreshTokenFamily{FamilyID: id}, nil
}
func (f *fRefresh) RevokeFamily(_ context.Context, _ string, _ domiam.TimeFn, _ domiam.RevokeReason) error {
	return nil
}
func (f *fRefresh) RevokeAllForUser(_ context.Context, _ string, _ domiam.TimeFn, _ domiam.RevokeReason) error {
	return nil
}
func (f *fRefresh) RevokeOne(_ context.Context, _ string, _ domiam.TimeFn, _ domiam.RevokeReason) error {
	return nil
}

type fHasher struct {
	mu sync.Mutex
	n  int
}

func (h *fHasher) Hash(_ context.Context, p string) (domiam.PasswordHash, error) {
	h.mu.Lock()
	h.n++
	n := h.n
	h.mu.Unlock()
	body := p + "::" + strconv.Itoa(n)
	if len(body) < 53 {
		body = body + ":xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	}
	return domiam.NewPasswordHash("$2a$10$" + body[:53])
}

func (h *fHasher) Compare(_ context.Context, hash domiam.PasswordHash, p string) error {
	if hash.IsZero() {
		return domiam.ErrInvalidCredentials
	}
	prefix := "$2a$10$" + p + "::"
	if len(hash.String()) < len(prefix) || hash.String()[:len(prefix)] != prefix {
		return domiam.ErrInvalidCredentials
	}
	return nil
}

func (h *fHasher) Strength(p string) error {
	if len(p) < 8 {
		return domiam.ErrPasswordTooWeak
	}
	return nil
}

type fSigner struct {
	mu          sync.Mutex
	tokToUserID map[string]string
}

func newSigner() *fSigner { return &fSigner{tokToUserID: map[string]string{}} }

func (s *fSigner) SignAccessToken(_ context.Context, uid string) (string, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp := time.Now().Add(15 * time.Minute).Unix()
	tok := "tok-" + uid + "-" + strconv.FormatInt(exp, 10)
	s.tokToUserID[tok] = uid
	return tok, exp, nil
}

func (s *fSigner) VerifyAccessToken(_ context.Context, tok string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	uid, ok := s.tokToUserID[tok]
	if !ok {
		return "", domiam.ErrUnauthorized
	}
	return uid, nil
}

type fMint struct {
	mu sync.Mutex
	n  int
}

func (m *fMint) Mint(_ context.Context) (string, string, error) {
	m.mu.Lock()
	m.n++
	n := m.n
	m.mu.Unlock()
	plain := "refresh-" + strconv.Itoa(n)
	sum := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(sum[:]), nil
}

type fRefHasher struct{}

func (fRefHasher) HashOpaque(p string) string {
	sum := sha256.Sum256([]byte(p))
	return hex.EncodeToString(sum[:])
}

type fUoW struct{}

func (fUoW) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fIAMEv struct{}

func (fIAMEv) Publish(_ context.Context, _ ...domiam.Event) error { return nil }

type fDecksRepo struct {
	mu  sync.Mutex
	rec map[string]*domdecks.Deck
}

func newDecks() *fDecksRepo { return &fDecksRepo{rec: map[string]*domdecks.Deck{}} }

func (f *fDecksRepo) Save(_ context.Context, d *domdecks.Deck) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec[d.ID()] = d
	return nil
}
func (f *fDecksRepo) ByID(_ context.Context, id string) (*domdecks.Deck, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.rec[id]
	if !ok {
		return nil, domdecks.ErrDeckNotFound
	}
	return d, nil
}
func (f *fDecksRepo) ByOwner(_ context.Context, o string, l int, _ string) ([]*domdecks.Deck, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*domdecks.Deck{}
	for _, d := range f.rec {
		if d.OwnerID() == o && !d.IsDeleted() {
			out = append(out, d)
		}
	}
	return out, "", nil
}
func (f *fDecksRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rec, id)
	return nil
}

func (f *fDecksRepo) OwnerAndCards(_ context.Context, deckID string) (string, []string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.rec[deckID]
	if !ok {
		return "", nil, domdecks.ErrDeckNotFound
	}
	ids := make([]string, 0, d.CardCount())
	for _, c := range d.Cards() {
		ids = append(ids, c.ID())
	}
	return d.OwnerID(), ids, nil
}

type fDecksEv struct{}

func (fDecksEv) Publish(_ context.Context, _ ...domdecks.Event) error { return nil }

type fSessions struct {
	mu  sync.Mutex
	rec map[string]*dompractice.Session
}

func newSessions() *fSessions { return &fSessions{rec: map[string]*dompractice.Session{}} }
func (f *fSessions) Save(_ context.Context, s *dompractice.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec[s.ID()] = s
	return nil
}
func (f *fSessions) ByID(_ context.Context, id string) (*dompractice.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.rec[id]
	if !ok {
		return nil, dompractice.ErrSessionNotFound
	}
	return s, nil
}
func (f *fSessions) LatestCompletedFor(_ context.Context, _, _ string) (*dompractice.Session, error) {
	return nil, dompractice.ErrSessionNotFound
}

type fProgress struct{}

func (fProgress) ByUserAndDeck(_ context.Context, _, _ string) (*dompractice.UserDeckProgress, error) {
	return nil, nil
}
func (fProgress) Save(_ context.Context, _ *dompractice.UserDeckProgress) error { return nil }
func (fProgress) DeleteByDeck(_ context.Context, _ string) error                { return nil }

type fPracticeEv struct{}

func (fPracticeEv) Publish(_ context.Context, _ ...dompractice.Event) error { return nil }

type fIdemStore struct {
	mu  sync.Mutex
	rec map[string]domiam.IdempotencyEntry
}

func newIdem() *fIdemStore { return &fIdemStore{rec: map[string]domiam.IdempotencyEntry{}} }

func (f *fIdemStore) Get(_ context.Context, scope, k string) (domiam.IdempotencyEntry, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rec[scope+"|"+k]
	return r, ok, nil
}

func (f *fIdemStore) Put(_ context.Context, e domiam.IdempotencyEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec[e.Scope+"|"+e.Key] = e
	return nil
}

// helpers --------------------------------------------------------------------

func buildRouter(t *testing.T) (http.Handler, *fSigner) {
	t.Helper()
	users := newU()
	refresh := newR()
	hasher := &fHasher{}
	signer := newSigner()
	mint := &fMint{}
	clk := clock.NewFixed(time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC))
	ids := idgen.NewSequential("rt")
	uow := fUoW{}
	iamEv := fIAMEv{}
	decksEv := fDecksEv{}
	practiceEv := fPracticeEv{}
	idem := newIdem()

	decksRepo := newDecks()
	sessions := newSessions()

	iamH := httpiam.New(httpiam.Deps{
		Register: appiam.RegisterUser{
			Users: users, RefreshTokens: refresh, Hasher: hasher, IDs: ids, Clock: clk,
			Tokens: mint, AccessSigner: signer, UoW: uow, Events: iamEv,
		},
		Login: appiam.LoginUser{
			Users: users, RefreshTokens: refresh, Hasher: hasher, IDs: ids, Clock: clk,
			Tokens: mint, AccessSigner: signer, UoW: uow, Events: iamEv,
		},
		Refresh: appiam.RefreshAccessToken{
			RefreshTokens: refresh, IDs: ids, Clock: clk, Tokens: mint,
			AccessSigner: signer, UoW: uow, Events: iamEv, Hasher: fRefHasher{},
		},
		Logout: appiam.LogoutUser{RefreshTokens: refresh, Hasher: fRefHasher{}, Clock: clk, UoW: uow, Events: iamEv},
		GetMe:  appiam.GetCurrentUser{Users: users},
		UpdateProfile:  appiam.UpdateProfile{Users: users, UoW: uow},
		ChangePassword: appiam.ChangePassword{Users: users, RefreshTokens: refresh, Hasher: hasher, Clock: clk, UoW: uow, Events: iamEv},
	})
	decksH := httpdecks.New(httpdecks.Deps{
		List: appdecks.ListUserDecks{Decks: decksRepo},
		Create: appdecks.CreateDeck{Decks: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: decksEv},
		Get: appdecks.GetDeck{Decks: decksRepo},
		Rename: appdecks.RenameDeck{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksEv},
		Delete: appdecks.DeleteDeck{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksEv},
		AddCard: appdecks.AddCard{Decks: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: decksEv},
		EditCard: appdecks.EditCard{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksEv},
		RemoveCard: appdecks.RemoveCard{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksEv},
		Reorder: appdecks.ReorderCards{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksEv},
		Generate: appdecks.GenerateDeckWithAI{
			AI: notConfiguredAI{}, IDs: ids, Clock: clk, Events: decksEv,
		},
		AIEnabled: false,
	})
	practiceH := httppractice.New(httppractice.Deps{
		Start: apppractice.StartSession{
			Sessions: sessions, Snapshots: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: practiceEv,
		},
		Rate: apppractice.RateCard{Sessions: sessions, Clock: clk, UoW: uow, Events: practiceEv},
		Finish: apppractice.FinishSession{Sessions: sessions, Clock: clk, UoW: uow, Events: practiceEv},
		Results: apppractice.GetResults{Sessions: sessions},
		Progress: apppractice.GetUserDeckProgress{Progress: fProgress{}},
	})

	authMW := middleware.NewAuth(signer)
	r := httproot.NewRouter(httproot.Deps{
		IAM: iamH, Decks: decksH, Practice: practiceH,
		Auth: authMW, Idem: idem, CORS: []string{"http://localhost:5173"},
	})
	return r, signer
}

type notConfiguredAI struct{}

func (notConfiguredAI) IsConfigured() bool { return false }
func (notConfiguredAI) Generate(_ context.Context, _ string) (domdecks.AIDeckDraft, error) {
	return domdecks.AIDeckDraft{}, domdecks.ErrAINotConfigured
}

// ---------------------------------------------------------------------------

func TestRouter_Healthz(t *testing.T) {
	r, _ := buildRouter(t)
	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	var b map[string]any
	require.NoError(t, json.Unmarshal(body, &b))
	require.Equal(t, "ok", b["status"])
}

func TestRouter_AuthRequiredRoutes_Reject401(t *testing.T) {
	r, _ := buildRouter(t)
	srv := httptest.NewServer(r)
	defer srv.Close()
	for _, p := range []string{"/api/me", "/api/decks", "/api/decks/x", "/api/decks/x/progress"} {
		resp, err := http.Get(srv.URL + p)
		require.NoError(t, err)
		require.Equal(t, 401, resp.StatusCode, "path %s", p)
		_ = resp.Body.Close()
	}
}

func TestRouter_RegisterLoginListCreateGet(t *testing.T) {
	r, _ := buildRouter(t)
	srv := httptest.NewServer(r)
	defer srv.Close()
	c := srv.Client()

	// Register
	resp := postJSON(t, c, srv.URL+"/api/auth/register",
		`{"email":"u@x.com","password":"supersecret","displayName":"U"}`, nil)
	require.Equal(t, 201, resp.StatusCode)
	var auth map[string]any
	decode(t, resp, &auth)
	tok := auth["accessToken"].(string)

	// List (empty)
	resp = getWithAuth(t, c, srv.URL+"/api/decks", tok)
	require.Equal(t, 200, resp.StatusCode)

	// Create
	resp = postJSONWithAuth(t, c, srv.URL+"/api/decks", `{"title":"D1"}`, tok, "k1")
	require.Equal(t, 201, resp.StatusCode)
	var d map[string]any
	decode(t, resp, &d)
	id := d["id"].(string)

	// Replay (idempotent)
	resp = postJSONWithAuth(t, c, srv.URL+"/api/decks", `{"title":"D1"}`, tok, "k1")
	require.Equal(t, 201, resp.StatusCode)
	require.Equal(t, "true", resp.Header.Get("Idempotent-Replay"))

	// Get
	resp = getWithAuth(t, c, srv.URL+"/api/decks/"+id, tok)
	require.Equal(t, 200, resp.StatusCode)

	// Delete
	req, _ := http.NewRequest("DELETE", srv.URL+"/api/decks/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestRouter_GenerateAIOff_Returns501(t *testing.T) {
	r, _ := buildRouter(t)
	srv := httptest.NewServer(r)
	defer srv.Close()
	c := srv.Client()
	resp := postJSON(t, c, srv.URL+"/api/auth/register",
		`{"email":"g@x.com","password":"supersecret","displayName":"G"}`, nil)
	var auth map[string]any
	decode(t, resp, &auth)
	tok := auth["accessToken"].(string)
	resp = postJSONWithAuth(t, c, srv.URL+"/api/decks/generate", `{"prompt":"hi"}`, tok, "kgen")
	require.Equal(t, 501, resp.StatusCode)
}

func TestRouter_PracticeFlow(t *testing.T) {
	r, _ := buildRouter(t)
	srv := httptest.NewServer(r)
	defer srv.Close()
	c := srv.Client()
	resp := postJSON(t, c, srv.URL+"/api/auth/register",
		`{"email":"p@x.com","password":"supersecret","displayName":"P"}`, nil)
	var auth map[string]any
	decode(t, resp, &auth)
	tok := auth["accessToken"].(string)

	// Create deck with two cards.
	resp = postJSONWithAuth(t, c, srv.URL+"/api/decks",
		`{"title":"P","cards":[{"term":"a","definition":"b"},{"term":"c","definition":"d"}]}`, tok, "kd")
	require.Equal(t, 201, resp.StatusCode)
	var deck map[string]any
	decode(t, resp, &deck)
	deckID := deck["id"].(string)
	cards := deck["cards"].([]any)
	c1 := cards[0].(map[string]any)["id"].(string)
	c2 := cards[1].(map[string]any)["id"].(string)

	// Start session.
	resp = postJSONWithAuth(t, c, srv.URL+"/api/practice/sessions",
		`{"deckId":"`+deckID+`","mode":"tracked"}`, tok, "ks")
	require.Equal(t, 201, resp.StatusCode)
	var s map[string]any
	decode(t, resp, &s)
	sid := s["id"].(string)

	// Rate both.
	for _, cid := range []string{c1, c2} {
		resp := postJSONWithAuth(t, c, srv.URL+"/api/practice/sessions/"+sid+"/ratings",
			`{"cardId":"`+cid+`","rating":2}`, tok, "")
		require.Equal(t, 200, resp.StatusCode)
	}

	// Finish.
	resp = postJSONWithAuth(t, c, srv.URL+"/api/practice/sessions/"+sid+"/finish", "", tok, "")
	require.Equal(t, 200, resp.StatusCode)

	// Results.
	resp = getWithAuth(t, c, srv.URL+"/api/practice/sessions/"+sid+"/results", tok)
	require.Equal(t, 200, resp.StatusCode)
	var res map[string]any
	decode(t, resp, &res)
	require.EqualValues(t, 2, res["countKnowKnow"])
}

// helpers --------------------------------------------------------------------

func postJSON(t *testing.T, c *http.Client, url, body string, _ map[string]string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	require.NoError(t, err)
	return resp
}

func postJSONWithAuth(t *testing.T, c *http.Client, url, body, tok, idemKey string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	resp, err := c.Do(req)
	require.NoError(t, err)
	return resp
}

func getWithAuth(t *testing.T, c *http.Client, url, tok string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.Do(req)
	require.NoError(t, err)
	return resp
}

func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}

