package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/micocards/api/internal/domain/iam"
)

// Store is the persistence port for the idempotency cache. The iam repo's
// IdempotencyKeys implementation satisfies it.
type Store interface {
	Get(ctx context.Context, scope, key string) (iam.IdempotencyEntry, bool, error)
	Put(ctx context.Context, e iam.IdempotencyEntry) error
}

// idempotencyTTL is the lifetime of a cached response (ADR 0005 — 24h).
const idempotencyTTL = 24 * time.Hour

// IdempotencyOptions controls one wrap of the middleware.
type IdempotencyOptions struct {
	Store    Store
	Required bool   // when true, missing header → 400
	Scope    string // canonical "<METHOD>:<route-pattern>"
}

// Idempotent wraps the next handler with read-through cache semantics on the
// Idempotency-Key header. On hit: reply with the cached status+body and the
// `Idempotent-Replay: true` header. On miss: run the inner handler, buffer
// its 2xx response, and persist before returning to the client.
//
// The wrap is a no-op when the request has no Idempotency-Key header AND
// Required=false (e.g. legacy clients).
func Idempotent(opts IdempotencyOptions) func(http.Handler) http.Handler {
	if opts.Store == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				if opts.Required {
					WriteError(w, r, iam.ErrIdempotencyKeyNeeded)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Read full body so we can hash it AND let the handler re-read it.
			body, err := io.ReadAll(r.Body)
			if err != nil {
				WriteError(w, r, &JSONDecodeError{Message: "read body: " + err.Error()})
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))

			scope := opts.Scope
			if scope == "" {
				scope = r.Method + ":" + r.URL.Path
			}
			reqHash := hashRequest(r, body)

			cached, ok, lookupErr := opts.Store.Get(r.Context(), scope, key)
			if lookupErr != nil {
				WriteError(w, r, lookupErr)
				return
			}
			if ok {
				if cached.RequestHash != reqHash {
					WriteError(w, r, iam.ErrIdempotencyConflict)
					return
				}
				w.Header().Set("Idempotent-Replay", "true")
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(cached.ResponseStatus)
				if len(cached.ResponseBody) > 0 {
					_, _ = w.Write(cached.ResponseBody)
				}
				return
			}

			rec := &bufferedResponse{ResponseWriter: w, body: &bytes.Buffer{}, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Only cache success-style responses (2xx). Errors (4xx/5xx) should
			// not poison the key — the client may retry with a fixed payload.
			if rec.status >= 200 && rec.status < 300 {
				entry := iam.IdempotencyEntry{
					Scope:          scope,
					Key:            key,
					RequestHash:    reqHash,
					ResponseStatus: rec.status,
					ResponseBody:   rec.body.Bytes(),
					ExpiresAtUnix:  time.Now().Add(idempotencyTTL).Unix(),
				}
				if putErr := opts.Store.Put(r.Context(), entry); putErr != nil &&
					!errors.Is(putErr, context.Canceled) {
					// Log via slog at the access-log layer; the response is
					// already sent so we cannot translate this into a 500.
				}
			}
		})
	}
}

// hashRequest computes a stable hash of method+path+query+body so the
// middleware can detect "same key, different request" (ADR 0005 → 409).
func hashRequest(r *http.Request, body []byte) string {
	h := sha256.New()
	h.Write([]byte(r.Method))
	h.Write([]byte("\n"))
	h.Write([]byte(r.URL.Path))
	h.Write([]byte("\n"))
	h.Write([]byte(r.URL.RawQuery))
	h.Write([]byte("\n"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// bufferedResponse captures the inner handler's output so we can mirror it
// to the cache and to the client without double-writing.
type bufferedResponse struct {
	http.ResponseWriter
	body        *bytes.Buffer
	status      int
	wroteHeader bool
}

// WriteHeader records the status and forwards.
func (b *bufferedResponse) WriteHeader(code int) {
	if b.wroteHeader {
		return
	}
	b.status = code
	b.wroteHeader = true
	b.ResponseWriter.WriteHeader(code)
}

// Write tees the body into the buffer.
func (b *bufferedResponse) Write(p []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	b.body.Write(p)
	return b.ResponseWriter.Write(p)
}
