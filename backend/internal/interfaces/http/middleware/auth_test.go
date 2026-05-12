package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/interfaces/http/middleware"
)

type fakeVerifier struct {
	userID string
	err    error
}

func (f fakeVerifier) VerifyAccessToken(_ context.Context, _ string) (string, error) {
	return f.userID, f.err
}

func TestAuth_Required_AcceptsBearer(t *testing.T) {
	a := middleware.NewAuth(fakeVerifier{userID: "u1"})
	called := false
	h := a.Required(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		require.Equal(t, "u1", middleware.UserIDFromContext(r.Context()))
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer abc.def")
	h.ServeHTTP(rr, req)
	require.True(t, called)
	require.Equal(t, 200, rr.Code)
}

func TestAuth_Required_RejectsMissingHeader(t *testing.T) {
	a := middleware.NewAuth(fakeVerifier{userID: "u1"})
	h := a.Required(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not be called")
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	require.Equal(t, 401, rr.Code)
}

func TestAuth_Required_RejectsInvalidToken(t *testing.T) {
	a := middleware.NewAuth(fakeVerifier{err: iam.ErrUnauthorized})
	h := a.Required(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not be called")
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer bad")
	h.ServeHTTP(rr, req)
	require.Equal(t, 401, rr.Code)
}

func TestAuth_Optional_PassesWithoutHeader(t *testing.T) {
	a := middleware.NewAuth(fakeVerifier{userID: "u1"})
	called := false
	h := a.Optional(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		require.Equal(t, "", middleware.UserIDFromContext(r.Context()))
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	require.True(t, called)
	require.Equal(t, 200, rr.Code)
}

func TestAuth_Optional_PopulatesWithValidHeader(t *testing.T) {
	a := middleware.NewAuth(fakeVerifier{userID: "u9"})
	h := a.Optional(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "u9", middleware.UserIDFromContext(r.Context()))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer good")
	h.ServeHTTP(rr, req)
}

func TestAuth_BearerCaseInsensitive(t *testing.T) {
	a := middleware.NewAuth(fakeVerifier{userID: "u1"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "bearer abc")
	a.Required(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})).ServeHTTP(rr, req)
	require.Equal(t, 200, rr.Code)
}
