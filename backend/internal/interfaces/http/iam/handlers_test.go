package iam_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appiam "github.com/micocards/api/internal/application/iam"
	domiam "github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
	httpiam "github.com/micocards/api/internal/interfaces/http/iam"
	"github.com/micocards/api/internal/interfaces/http/middleware"
)

// ----- inline fakes (no test-only files leaking outside the package) -----

type fakeUsers struct {
	mu      sync.Mutex
	byID    map[string]*domiam.User
	byEmail map[string]*domiam.User
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byID: map[string]*domiam.User{}, byEmail: map[string]*domiam.User{}}
}

func (f *fakeUsers) Save(_ context.Context, u *domiam.User) error {
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

func (f *fakeUsers) ByID(_ context.Context, id string) (*domiam.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domiam.ErrUserNotFound
}

func (f *fakeUsers) ByEmail(_ context.Context, e domiam.EmailAddress) (*domiam.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.byEmail[e.String()]; ok {
		return u, nil
	}
	return nil, domiam.ErrUserNotFound
}

func (f *fakeUsers) EmailExists(_ context.Context, e domiam.EmailAddress) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.byEmail[e.String()]
	return ok, nil
}

type fakeRefresh struct {
	mu     sync.Mutex
	byID   map[string]domiam.RefreshToken
	byHash map[string]string
}

func newFakeRefresh() *fakeRefresh {
	return &fakeRefresh{byID: map[string]domiam.RefreshToken{}, byHash: map[string]string{}}
}

func (f *fakeRefresh) Save(_ context.Context, t domiam.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[t.ID] = t
	f.byHash[t.OpaqueHash] = t.ID
	return nil
}

func (f *fakeRefresh) ByOpaqueHash(_ context.Context, h string) (domiam.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byHash[h]
	if !ok {
		return domiam.RefreshToken{}, domiam.ErrRefreshTokenInvalid
	}
	return f.byID[id], nil
}
func (f *fakeRefresh) FamilyByID(_ context.Context, id string) (domiam.RefreshTokenFamily, error) {
	return domiam.RefreshTokenFamily{FamilyID: id}, nil
}

func (f *fakeRefresh) RevokeFamily(_ context.Context, fam string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	for id, t := range f.byID {
		if t.FamilyID != fam || t.RevokedAt != nil {
			continue
		}
		tt := now
		t.RevokedAt = &tt
		t.RevokeNote = reason
		f.byID[id] = t
	}
	return nil
}

func (f *fakeRefresh) RevokeAllForUser(_ context.Context, uid string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	for id, t := range f.byID {
		if t.UserID != uid || t.RevokedAt != nil {
			continue
		}
		tt := now
		t.RevokedAt = &tt
		t.RevokeNote = reason
		f.byID[id] = t
	}
	return nil
}

func (f *fakeRefresh) RevokeOne(_ context.Context, id string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[id]
	if !ok {
		return domiam.ErrRefreshTokenInvalid
	}
	if t.RevokedAt != nil {
		return nil
	}
	now := time.Now().UTC()
	t.RevokedAt = &now
	t.RevokeNote = reason
	f.byID[id] = t
	return nil
}

type fakeHasher struct {
	mu sync.Mutex
	n  int
}

func (h *fakeHasher) Hash(_ context.Context, p string) (domiam.PasswordHash, error) {
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

func (h *fakeHasher) Compare(_ context.Context, hash domiam.PasswordHash, p string) error {
	if hash.IsZero() {
		return domiam.ErrInvalidCredentials
	}
	prefix := "$2a$10$" + p + "::"
	if len(hash.String()) < len(prefix) || hash.String()[:len(prefix)] != prefix {
		return domiam.ErrInvalidCredentials
	}
	return nil
}

func (h *fakeHasher) Strength(p string) error {
	if len(p) < 8 {
		return domiam.ErrPasswordTooWeak
	}
	return nil
}

type fakeAccess struct{}

func (fakeAccess) SignAccessToken(_ context.Context, uid string) (string, int64, error) {
	exp := time.Now().Add(15 * time.Minute).Unix()
	return "access-" + uid + "-" + strconv.FormatInt(exp, 10), exp, nil
}

func (fakeAccess) VerifyAccessToken(_ context.Context, tok string) (string, error) {
	if tok == "" {
		return "", domiam.ErrUnauthorized
	}
	return "u-from-token", nil
}

type fakeMint struct {
	mu sync.Mutex
	n  int
}

func (m *fakeMint) Mint(_ context.Context) (string, string, error) {
	m.mu.Lock()
	m.n++
	n := m.n
	m.mu.Unlock()
	plain := "refresh-" + strconv.Itoa(n)
	sum := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(sum[:]), nil
}

type fakeRefreshHasher struct{}

func (fakeRefreshHasher) HashOpaque(p string) string {
	sum := sha256.Sum256([]byte(p))
	return hex.EncodeToString(sum[:])
}

type fakeUoW struct{}

func (fakeUoW) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type fakeEvents struct{}

func (fakeEvents) Publish(_ context.Context, _ ...domiam.Event) error { return nil }

// ----- helpers -----

func buildHandlers(t *testing.T) (*httpiam.Handlers, *fakeUsers) {
	t.Helper()
	users := newFakeUsers()
	refresh := newFakeRefresh()
	hasher := &fakeHasher{}
	clk := clock.NewFixed(time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC))
	ids := idgen.NewSequential("id")
	mint := &fakeMint{}
	signer := fakeAccess{}
	uow := fakeUoW{}
	events := fakeEvents{}
	return httpiam.New(httpiam.Deps{
		Register: appiam.RegisterUser{
			Users: users, RefreshTokens: refresh, Hasher: hasher, IDs: ids, Clock: clk,
			Tokens: mint, AccessSigner: signer, UoW: uow, Events: events,
		},
		Login: appiam.LoginUser{
			Users: users, RefreshTokens: refresh, Hasher: hasher, IDs: ids, Clock: clk,
			Tokens: mint, AccessSigner: signer, UoW: uow, Events: events,
		},
		Refresh: appiam.RefreshAccessToken{
			RefreshTokens: refresh, IDs: ids, Clock: clk, Tokens: mint,
			AccessSigner: signer, UoW: uow, Events: events, Hasher: fakeRefreshHasher{},
		},
		Logout: appiam.LogoutUser{
			RefreshTokens: refresh, Hasher: fakeRefreshHasher{}, Clock: clk, UoW: uow, Events: events,
		},
		GetMe:         appiam.GetCurrentUser{Users: users},
		UpdateProfile: appiam.UpdateProfile{Users: users, UoW: uow},
		ChangePassword: appiam.ChangePassword{
			Users: users, RefreshTokens: refresh, Hasher: hasher, Clock: clk, UoW: uow, Events: events,
		},
	}), users
}

func newAuthMW() *middleware.Auth {
	return middleware.NewAuth(fakeAccess{})
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// ----- tests -----

func TestRegister_201(t *testing.T) {
	h, _ := buildHandlers(t)
	rr := doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register",
		map[string]any{"email": "user@example.com", "password": "supersecret", "displayName": "User"}, nil)
	require.Equal(t, 201, rr.Code, rr.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.NotEmpty(t, body["accessToken"])
	require.NotEmpty(t, body["refreshToken"])
}

func TestRegister_422_MissingFields(t *testing.T) {
	h, _ := buildHandlers(t)
	rr := doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register", map[string]any{}, nil)
	require.Equal(t, 422, rr.Code)
}

func TestRegister_409_EmailTaken(t *testing.T) {
	h, _ := buildHandlers(t)
	body := map[string]any{"email": "x@x.com", "password": "supersecret", "displayName": "X"}
	doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register", body, nil)
	rr := doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register", body, nil)
	require.Equal(t, 409, rr.Code)
}

func TestRegister_400_BadJSON(t *testing.T) {
	h, _ := buildHandlers(t)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader([]byte("{not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Register).ServeHTTP(rr, req)
	require.Equal(t, 400, rr.Code)
}

func TestLogin_200_AndBadCreds(t *testing.T) {
	h, _ := buildHandlers(t)
	doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register",
		map[string]any{"email": "a@b.com", "password": "supersecret", "displayName": "A"}, nil)
	rr := doJSON(t, http.HandlerFunc(h.Login), "POST", "/api/auth/login",
		map[string]any{"email": "a@b.com", "password": "supersecret"}, nil)
	require.Equal(t, 200, rr.Code)
	rr = doJSON(t, http.HandlerFunc(h.Login), "POST", "/api/auth/login",
		map[string]any{"email": "a@b.com", "password": "wrongpassx"}, nil)
	require.Equal(t, 401, rr.Code)
}

func TestLogin_422_Missing(t *testing.T) {
	h, _ := buildHandlers(t)
	rr := doJSON(t, http.HandlerFunc(h.Login), "POST", "/api/auth/login", map[string]any{}, nil)
	require.Equal(t, 422, rr.Code)
}

func TestRefresh_200_AndInvalid(t *testing.T) {
	h, _ := buildHandlers(t)
	rrReg := doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register",
		map[string]any{"email": "r@x.com", "password": "supersecret", "displayName": "R"}, nil)
	var reg map[string]any
	require.NoError(t, json.Unmarshal(rrReg.Body.Bytes(), &reg))
	tok, _ := reg["refreshToken"].(string)
	rr := doJSON(t, http.HandlerFunc(h.Refresh), "POST", "/api/auth/refresh",
		map[string]any{"refreshToken": tok}, nil)
	require.Equal(t, 200, rr.Code)
	rr = doJSON(t, http.HandlerFunc(h.Refresh), "POST", "/api/auth/refresh",
		map[string]any{"refreshToken": "nope"}, nil)
	require.Equal(t, 401, rr.Code)
}

func TestGetMe_RequiresAuth(t *testing.T) {
	h, _ := buildHandlers(t)
	authMW := newAuthMW()
	wrapped := authMW.Required(http.HandlerFunc(h.GetMe))
	rr := doJSON(t, wrapped, "GET", "/api/me", nil, nil)
	require.Equal(t, 401, rr.Code)
}

func TestGetMe_404_WhenUserMissing(t *testing.T) {
	h, _ := buildHandlers(t)
	authMW := newAuthMW()
	wrapped := authMW.Required(http.HandlerFunc(h.GetMe))
	rr := doJSON(t, wrapped, "GET", "/api/me", nil, map[string]string{"Authorization": "Bearer xxx"})
	// fakeAccess returns "u-from-token" — the user does not exist in the fake repo → 404.
	require.Equal(t, 404, rr.Code)
}

func TestGetMe_200_AfterRegister(t *testing.T) {
	// Register + then GetMe with a context that already has the user id.
	h, users := buildHandlers(t)
	doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register",
		map[string]any{"email": "m@x.com", "password": "supersecret", "displayName": "M"}, nil)
	require.Equal(t, 1, len(users.byID))
	var firstID string
	for k := range users.byID {
		firstID = k
		break
	}
	// Inject userID into context manually.
	req := httptest.NewRequest("GET", "/api/me", nil)
	req = req.WithContext(middleware.WithUserID(req.Context(), firstID))
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.GetMe).ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code, rr.Body.String())
}

func TestChangePassword_422_Empty(t *testing.T) {
	h, _ := buildHandlers(t)
	req := httptest.NewRequest("POST", "/api/auth/change-password",
		bytes.NewReader([]byte(`{"currentPassword":"","newPassword":""}`)))
	req = req.WithContext(middleware.WithUserID(req.Context(), "u1"))
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.ChangePassword).ServeHTTP(rr, req)
	require.Equal(t, 422, rr.Code)
}

func TestUpdateMe_RequiresAuth(t *testing.T) {
	h, _ := buildHandlers(t)
	wrapped := newAuthMW().Required(http.HandlerFunc(h.UpdateMe))
	rr := doJSON(t, wrapped, "PUT", "/api/me", map[string]any{"displayName": "X"}, nil)
	require.Equal(t, 401, rr.Code)
}

func TestLogout_OK_Idempotent(t *testing.T) {
	h, _ := buildHandlers(t)
	rr := doJSON(t, http.HandlerFunc(h.Logout), "POST", "/api/auth/logout",
		map[string]any{"refreshToken": "anything"}, nil)
	require.Equal(t, 200, rr.Code)
}

func TestHealthz(t *testing.T) {
	h, _ := buildHandlers(t)
	rr := doJSON(t, http.HandlerFunc(h.Healthz), "GET", "/api/healthz", nil, nil)
	require.Equal(t, 200, rr.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, "ok", body["status"])
}

func TestAvatar_501(t *testing.T) {
	h, _ := buildHandlers(t)
	rr := doJSON(t, http.HandlerFunc(h.Avatar), "POST", "/api/me/avatar", nil, nil)
	require.Equal(t, 501, rr.Code)
}

func TestUpdateMe_200(t *testing.T) {
	h, users := buildHandlers(t)
	doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register",
		map[string]any{"email": "up@x.com", "password": "supersecret", "displayName": "Up"}, nil)
	var firstID string
	for k := range users.byID {
		firstID = k
		break
	}
	newName := "Updated"
	body := map[string]any{"displayName": newName}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/api/me", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithUserID(req.Context(), firstID))
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.UpdateMe).ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code, rr.Body.String())
}

func TestUpdateMe_BadJSON(t *testing.T) {
	h, _ := buildHandlers(t)
	req := httptest.NewRequest("PUT", "/api/me", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithUserID(req.Context(), "u"))
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.UpdateMe).ServeHTTP(rr, req)
	require.Equal(t, 400, rr.Code)
}

func TestChangePassword_200(t *testing.T) {
	h, users := buildHandlers(t)
	doJSON(t, http.HandlerFunc(h.Register), "POST", "/api/auth/register",
		map[string]any{"email": "cp@x.com", "password": "supersecret", "displayName": "CP"}, nil)
	var firstID string
	for k := range users.byID {
		firstID = k
		break
	}
	body, _ := json.Marshal(map[string]any{
		"currentPassword": "supersecret", "newPassword": "newsecret11",
	})
	req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithUserID(req.Context(), firstID))
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.ChangePassword).ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code, rr.Body.String())
}

func TestChangePassword_BadJSON(t *testing.T) {
	h, _ := buildHandlers(t)
	req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithUserID(req.Context(), "u"))
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.ChangePassword).ServeHTTP(rr, req)
	require.Equal(t, 400, rr.Code)
}

func TestLogin_BadJSON(t *testing.T) {
	h, _ := buildHandlers(t)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Login).ServeHTTP(rr, req)
	require.Equal(t, 400, rr.Code)
}

func TestRefresh_400_BadJSON(t *testing.T) {
	h, _ := buildHandlers(t)
	req := httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Refresh).ServeHTTP(rr, req)
	require.Equal(t, 400, rr.Code)
}

func TestRefresh_401_EmptyToken(t *testing.T) {
	h, _ := buildHandlers(t)
	rr := doJSON(t, http.HandlerFunc(h.Refresh), "POST", "/api/auth/refresh",
		map[string]any{"refreshToken": ""}, nil)
	require.Equal(t, 401, rr.Code)
}

func TestLogout_400_BadJSON(t *testing.T) {
	h, _ := buildHandlers(t)
	req := httptest.NewRequest("POST", "/api/auth/logout", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	http.HandlerFunc(h.Logout).ServeHTTP(rr, req)
	require.Equal(t, 400, rr.Code)
}
