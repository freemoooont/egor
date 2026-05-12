package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/interfaces/http/middleware"
)

func TestAccessLog_LogsLine(t *testing.T) {
	buf := &bytes.Buffer{}
	log := slog.New(slog.NewJSONHandler(buf, nil))
	mw := middleware.AccessLog(log)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
		_, _ = w.Write([]byte("ok"))
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	require.Equal(t, 204, rr.Code)
	require.Contains(t, buf.String(), `"status":204`)
	require.Contains(t, buf.String(), `"method":"GET"`)
}

func TestCORS_AllowedOrigin(t *testing.T) {
	mw := middleware.CORS([]string{"http://localhost:5173"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("OPTIONS", "/x", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, 204, rr.Code)
	require.Equal(t, "http://localhost:5173", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	mw := middleware.CORS([]string{"http://localhost:5173"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "http://evil.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, "", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestChain_OrderOutsideIn(t *testing.T) {
	order := []string{}
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "1in")
			next.ServeHTTP(w, r)
			order = append(order, "1out")
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "2in")
			next.ServeHTTP(w, r)
			order = append(order, "2out")
		})
	}
	h := middleware.Chain(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		order = append(order, "core")
	}), mw1, mw2)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	require.Equal(t, []string{"1in", "2in", "core", "2out", "1out"}, order)
}

func TestDecodeStrictJSON_RejectsUnknownFields(t *testing.T) {
	type T struct {
		A int `json:"a"`
	}
	req := httptest.NewRequest("POST", "/x", bytes.NewBufferString(`{"a":1,"b":2}`))
	req.Header.Set("Content-Type", "application/json")
	var v T
	err := middleware.DecodeStrictJSON(req, &v)
	require.Error(t, err)
}

func TestDecodeStrictJSON_OK(t *testing.T) {
	type T struct {
		A int `json:"a"`
	}
	req := httptest.NewRequest("POST", "/x", bytes.NewBufferString(`{"a":1}`))
	var v T
	require.NoError(t, middleware.DecodeStrictJSON(req, &v))
	require.Equal(t, 1, v.A)
}

func TestDecodeStrictJSON_EmptyBody(t *testing.T) {
	type T struct {
		A int `json:"a"`
	}
	req := httptest.NewRequest("POST", "/x", bytes.NewBufferString(""))
	var v T
	err := middleware.DecodeStrictJSON(req, &v)
	require.Error(t, err)
}

func TestDecodeStrictJSON_TypeMismatch(t *testing.T) {
	type T struct {
		A int `json:"a"`
	}
	req := httptest.NewRequest("POST", "/x", bytes.NewBufferString(`{"a":"x"}`))
	var v T
	err := middleware.DecodeStrictJSON(req, &v)
	require.Error(t, err)
}
