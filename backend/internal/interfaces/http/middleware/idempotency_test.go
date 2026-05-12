package middleware_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/interfaces/http/middleware"
)

type fakeStore struct {
	mu    sync.Mutex
	rows  map[string]iam.IdempotencyEntry
	getEr error
	putEr error
}

func newFakeStore() *fakeStore { return &fakeStore{rows: map[string]iam.IdempotencyEntry{}} }

func key(scope, k string) string { return scope + "|" + k }

func (f *fakeStore) Get(_ context.Context, scope, k string) (iam.IdempotencyEntry, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getEr != nil {
		return iam.IdempotencyEntry{}, false, f.getEr
	}
	r, ok := f.rows[key(scope, k)]
	return r, ok, nil
}

func (f *fakeStore) Put(_ context.Context, e iam.IdempotencyEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.putEr != nil {
		return f.putEr
	}
	f.rows[key(e.Scope, e.Key)] = e
	return nil
}

func TestIdempotency_NoHeader_PassesThrough(t *testing.T) {
	store := newFakeStore()
	mw := middleware.Idempotent(middleware.IdempotencyOptions{Store: store, Scope: "POST:/x"})
	calls := 0
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/x", strings.NewReader("{}"))
	h.ServeHTTP(rr, req)
	require.Equal(t, 201, rr.Code)
	require.Equal(t, 1, calls)
}

func TestIdempotency_FirstCallPersistsThenReplay(t *testing.T) {
	store := newFakeStore()
	mw := middleware.Idempotent(middleware.IdempotencyOptions{Store: store, Scope: "POST:/x"})
	calls := 0
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	body := strings.NewReader(`{"a":1}`)
	req1 := httptest.NewRequest("POST", "/x", body)
	req1.Header.Set("Idempotency-Key", "k1")
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)
	require.Equal(t, 201, rr1.Code)
	require.Equal(t, 1, calls)

	req2 := httptest.NewRequest("POST", "/x", strings.NewReader(`{"a":1}`))
	req2.Header.Set("Idempotency-Key", "k1")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	require.Equal(t, 201, rr2.Code)
	require.Equal(t, "true", rr2.Header().Get("Idempotent-Replay"))
	require.Equal(t, 1, calls, "handler should not be called on replay")
	require.JSONEq(t, `{"id":"1"}`, rr2.Body.String())
}

func TestIdempotency_ConflictingRequest(t *testing.T) {
	store := newFakeStore()
	mw := middleware.Idempotent(middleware.IdempotencyOptions{Store: store, Scope: "POST:/x"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{}`))
	}))
	{
		req := httptest.NewRequest("POST", "/x", strings.NewReader(`{"a":1}`))
		req.Header.Set("Idempotency-Key", "k1")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, 201, rr.Code)
	}
	req2 := httptest.NewRequest("POST", "/x", strings.NewReader(`{"a":2}`))
	req2.Header.Set("Idempotency-Key", "k1")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	require.Equal(t, 409, rr2.Code)
}

func TestIdempotency_RequiredButMissingHeader(t *testing.T) {
	store := newFakeStore()
	mw := middleware.Idempotent(middleware.IdempotencyOptions{Store: store, Required: true, Scope: "POST:/x"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not be called")
	}))
	req := httptest.NewRequest("POST", "/x", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, 400, rr.Code)
}

func TestIdempotency_NoStore_PassesThrough(t *testing.T) {
	mw := middleware.Idempotent(middleware.IdempotencyOptions{})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(201)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/x", strings.NewReader(`{}`))
	req.Header.Set("Idempotency-Key", "k1")
	h.ServeHTTP(rr, req)
	require.Equal(t, 201, rr.Code)
}

func TestIdempotency_BodyAvailableToHandler(t *testing.T) {
	store := newFakeStore()
	mw := middleware.Idempotent(middleware.IdempotencyOptions{Store: store, Scope: "POST:/x"})
	var got string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(201)
	}))
	req := httptest.NewRequest("POST", "/x", strings.NewReader(`{"hi":1}`))
	req.Header.Set("Idempotency-Key", "k")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, `{"hi":1}`, got)
}
