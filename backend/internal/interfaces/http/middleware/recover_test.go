package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/interfaces/http/middleware"
)

func TestRecover_TranslatesPanicTo500(t *testing.T) {
	mw := middleware.Recover(slog.Default())
	h := mw(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	require.Equal(t, 500, rr.Code)
}
