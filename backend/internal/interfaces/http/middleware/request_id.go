package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// requestIDKey is the context key under which the per-request id is stored.
type requestIDKey struct{}

// RequestIDFromContext returns the X-Request-ID for the current request, or "".
func RequestIDFromContext(ctx context.Context) string {
	v := ctx.Value(requestIDKey{})
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// withRequestID attaches the id to the request's context.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID is the middleware constructor. It reuses an inbound X-Request-ID
// header when present (so log-tracing across hops works); otherwise it mints
// a fresh 16-byte hex id.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = mintID()
			}
			w.Header().Set("X-Request-ID", id)
			ctx := withRequestID(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func mintID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
