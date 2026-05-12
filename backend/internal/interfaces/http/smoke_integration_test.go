//go:build integration

// Package http_test smoke covers the full router against a real Postgres
// (testcontainers). Tagged `integration` so unit-test runs skip it.
package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appdecks "github.com/micocards/api/internal/application/decks"
	appiam "github.com/micocards/api/internal/application/iam"
	apppractice "github.com/micocards/api/internal/application/practice"
	"github.com/micocards/api/internal/infrastructure/ai"
	"github.com/micocards/api/internal/infrastructure/auth"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/events"
	"github.com/micocards/api/internal/infrastructure/idgen"
	pgsql "github.com/micocards/api/internal/infrastructure/postgres"
	"github.com/micocards/api/internal/infrastructure/postgres/decksrepo"
	"github.com/micocards/api/internal/infrastructure/postgres/iamrepo"
	"github.com/micocards/api/internal/infrastructure/postgres/outbox"
	"github.com/micocards/api/internal/infrastructure/postgres/practicerepo"
	"github.com/micocards/api/internal/infrastructure/postgres/testdb"
	httproot "github.com/micocards/api/internal/interfaces/http"
	httpdecks "github.com/micocards/api/internal/interfaces/http/decks"
	httpiam "github.com/micocards/api/internal/interfaces/http/iam"
	"github.com/micocards/api/internal/interfaces/http/middleware"
	httppractice "github.com/micocards/api/internal/interfaces/http/practice"
)

// TestEndToEndSmoke runs register → login → create deck w/ 2 cards → start
// session → rate two cards → finish → results, all against a real Postgres
// container booted by testdb.New.
func TestEndToEndSmoke(t *testing.T) {
	pool, _ := testdb.New(t)
	testdb.CleanAll(t, t.Context(), pool)

	signer, err := auth.NewAccessTokenSigner([]byte("test-jwt-secret-32-byte-length-min!!!"), 15*time.Minute)
	require.NoError(t, err)
	hasher := auth.NewBcryptHasher(4, 8) // low cost speeds up tests
	mint := auth.NewRefreshMinter()
	clk := clock.System{}
	ids := idgen.UUID{}
	uow := pgsql.NewUnitOfWork(pool)

	bus := events.NewPublisher()
	iamOB, _ := outbox.New(pool, "iam")
	decksOB, _ := outbox.New(pool, "decks")
	practiceOB, _ := outbox.New(pool, "practice")
	iamPub := events.NewIAMPublisher(bus, iamOB)
	decksPub := events.NewDecksPublisher(bus, decksOB)
	practicePub := events.NewPracticePublisher(bus, practiceOB)

	users := iamrepo.NewUsers(pool)
	refreshes := iamrepo.NewRefreshTokens(pool, nil)
	idem := iamrepo.NewIdempotencyKeys(pool)
	decksRepo := decksrepo.NewDecks(pool)
	sessRepo := practicerepo.NewSessions(pool)
	progRepo := practicerepo.NewUserDeckProgresses(pool)

	iamH := httpiam.New(httpiam.Deps{
		Register: appiam.RegisterUser{Users: users, RefreshTokens: refreshes, Hasher: hasher, IDs: ids, Clock: clk, Tokens: mint, AccessSigner: signer, UoW: uow, Events: iamPub},
		Login:    appiam.LoginUser{Users: users, RefreshTokens: refreshes, Hasher: hasher, IDs: ids, Clock: clk, Tokens: mint, AccessSigner: signer, UoW: uow, Events: iamPub},
		Refresh:  appiam.RefreshAccessToken{RefreshTokens: refreshes, IDs: ids, Clock: clk, Tokens: mint, AccessSigner: signer, UoW: uow, Events: iamPub, Hasher: refreshHasherShim{}},
		Logout:   appiam.LogoutUser{RefreshTokens: refreshes, Hasher: refreshHasherShim{}, Clock: clk, UoW: uow, Events: iamPub},
		GetMe:    appiam.GetCurrentUser{Users: users},
		UpdateProfile:  appiam.UpdateProfile{Users: users, UoW: uow},
		ChangePassword: appiam.ChangePassword{Users: users, RefreshTokens: refreshes, Hasher: hasher, Clock: clk, UoW: uow, Events: iamPub},
		DBPool:   pool,
	})
	decksH := httpdecks.New(httpdecks.Deps{
		List:       appdecks.ListUserDecks{Decks: decksRepo},
		Create:     appdecks.CreateDeck{Decks: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: decksPub},
		Get:        appdecks.GetDeck{Decks: decksRepo},
		Rename:     appdecks.RenameDeck{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		Delete:     appdecks.DeleteDeck{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		AddCard:    appdecks.AddCard{Decks: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: decksPub},
		EditCard:   appdecks.EditCard{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		RemoveCard: appdecks.RemoveCard{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		Reorder:    appdecks.ReorderCards{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		Generate:   appdecks.GenerateDeckWithAI{AI: ai.NotConfigured{}, IDs: ids, Clock: clk, Events: decksPub},
	})
	practiceH := httppractice.New(httppractice.Deps{
		Start:    apppractice.StartSession{Sessions: sessRepo, Snapshots: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: practicePub},
		Rate:     apppractice.RateCard{Sessions: sessRepo, Clock: clk, UoW: uow, Events: practicePub},
		Finish:   apppractice.FinishSession{Sessions: sessRepo, Clock: clk, UoW: uow, Events: practicePub},
		Results:  apppractice.GetResults{Sessions: sessRepo},
		Progress: apppractice.GetUserDeckProgress{Progress: progRepo},
	})

	authMW := middleware.NewAuth(signer)
	router := httproot.NewRouter(httproot.Deps{
		IAM: iamH, Decks: decksH, Practice: practiceH,
		Auth: authMW, Idem: idem, CORS: []string{"http://localhost:5173"}, DBPool: pool,
	})
	srv := httptest.NewServer(router)
	defer srv.Close()
	c := srv.Client()

	// 1. Healthz.
	resp, err := c.Get(srv.URL + "/api/healthz")
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	_ = resp.Body.Close()

	// 2. Register a new user.
	resp = postJSONRaw(t, c, srv.URL+"/api/auth/register",
		`{"email":"smoke@x.com","password":"supersecret","displayName":"S"}`, "ksmoke-reg", "")
	require.Equal(t, 201, resp.StatusCode, mustBody(resp))
	var auth1 map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&auth1))
	resp.Body.Close()
	tok := auth1["accessToken"].(string)

	// 3. Login again to mint a fresh pair (verifies LoginUser path).
	resp = postJSONRaw(t, c, srv.URL+"/api/auth/login",
		`{"email":"smoke@x.com","password":"supersecret"}`, "", "")
	require.Equal(t, 200, resp.StatusCode, mustBody(resp))
	resp.Body.Close()

	// 4. Create deck with two cards.
	body := `{"title":"Smoke Deck","cards":[{"term":"hi","definition":"hello"},{"term":"bye","definition":"goodbye"}]}`
	resp = postJSONRaw(t, c, srv.URL+"/api/decks", body, "ksmoke-deck", tok)
	require.Equal(t, 201, resp.StatusCode, mustBody(resp))
	var deck map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&deck))
	resp.Body.Close()
	deckID := deck["id"].(string)
	cards := deck["cards"].([]any)
	c1 := cards[0].(map[string]any)["id"].(string)
	c2 := cards[1].(map[string]any)["id"].(string)

	// 5. Start a tracked practice session.
	resp = postJSONRaw(t, c, srv.URL+"/api/practice/sessions",
		`{"deckId":"`+deckID+`","mode":"tracked"}`, "ksmoke-sess", tok)
	require.Equal(t, 201, resp.StatusCode, mustBody(resp))
	var sess map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&sess))
	resp.Body.Close()
	sid := sess["id"].(string)

	// 6. Rate both cards.
	for _, cid := range []string{c1, c2} {
		resp := postJSONRaw(t, c, srv.URL+"/api/practice/sessions/"+sid+"/ratings",
			`{"cardId":"`+cid+`","rating":2}`, "", tok)
		require.Equal(t, 200, resp.StatusCode, mustBody(resp))
		resp.Body.Close()
	}

	// 7. Finish.
	resp = postJSONRaw(t, c, srv.URL+"/api/practice/sessions/"+sid+"/finish", "", "", tok)
	require.Equal(t, 200, resp.StatusCode, mustBody(resp))
	resp.Body.Close()

	// 8. Fetch results.
	resp, err = httpGetWithToken(c, srv.URL+"/api/practice/sessions/"+sid+"/results", tok)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode, mustBody(resp))
	var res map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&res))
	resp.Body.Close()
	require.EqualValues(t, 2, res["countKnowKnow"])

	// 9. List my decks (should contain the smoke deck).
	resp, err = httpGetWithToken(c, srv.URL+"/api/decks", tok)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	var list map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	resp.Body.Close()
	require.NotEmpty(t, list["decks"])
}

func postJSONRaw(t *testing.T, c *http.Client, url, body, idemKey, tok string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	require.NoError(t, err)
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

func httpGetWithToken(c *http.Client, url, tok string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	return c.Do(req)
}

func mustBody(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(strings.NewReader(string(b)))
	return string(b)
}

type refreshHasherShim struct{}

func (refreshHasherShim) HashOpaque(p string) string { return auth.HashOpaque(p) }
