package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/interfaces/http/middleware"
)

func TestRequestID_ReusesInbound(t *testing.T) {
	mw := middleware.RequestID()
	var got string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = middleware.RequestIDFromContext(r.Context())
	}))
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Request-ID", "abcd1234")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, "abcd1234", got)
	require.Equal(t, "abcd1234", rr.Header().Get("X-Request-ID"))
}

func TestRequestID_MintsWhenMissing(t *testing.T) {
	mw := middleware.RequestID()
	var got string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = middleware.RequestIDFromContext(r.Context())
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	require.NotEmpty(t, got)
	require.Equal(t, got, rr.Header().Get("X-Request-ID"))
}
