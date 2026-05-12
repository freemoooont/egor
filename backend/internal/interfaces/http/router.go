// Package http composes the HTTP transport for the Micocards API. The router
// builds a single http.ServeMux, wraps each protected route with the auth
// middleware, attaches Idempotency-Key handling to the non-idempotent POSTs
// per ADR 0005, and prepends the global middleware stack: request-id →
// recover → CORS → access-log → error-mapping (per-route).
package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	httpdecks "github.com/micocards/api/internal/interfaces/http/decks"
	httpiam "github.com/micocards/api/internal/interfaces/http/iam"
	"github.com/micocards/api/internal/interfaces/http/middleware"
	httppractice "github.com/micocards/api/internal/interfaces/http/practice"
)

// Deps groups every dependency the router needs at construction time.
type Deps struct {
	IAM       *httpiam.Handlers
	Decks     *httpdecks.Handlers
	Practice  *httppractice.Handlers
	Auth      *middleware.Auth
	Idem      middleware.Store
	Logger    *slog.Logger
	CORS      []string
	DBPool    *pgxpool.Pool
	StartTime time.Time
}

// NewRouter returns the configured ServeMux for the API.
func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	registerHealth(mux, d)
	registerAuth(mux, d)
	registerMe(mux, d)
	registerDecks(mux, d)
	registerPractice(mux, d)

	handler := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.Recover(d.Logger),
		middleware.CORS(d.CORS),
		middleware.AccessLog(d.Logger),
	)
	return handler
}

func registerHealth(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET /api/healthz", d.IAM.Healthz)
}

func registerAuth(mux *http.ServeMux, d Deps) {
	registerMaybeIdempotent := func(method, pattern string, handler http.HandlerFunc) {
		// Wrap with idempotency middleware (Required=false → header optional).
		wrapped := middleware.Idempotent(middleware.IdempotencyOptions{
			Store: d.Idem, Scope: method + ":" + pattern,
		})(handler)
		mux.Handle(method+" "+pattern, wrapped)
	}
	registerMaybeIdempotent(http.MethodPost, "/api/auth/register", d.IAM.Register)
	mux.HandleFunc("POST /api/auth/login", d.IAM.Login)
	mux.HandleFunc("POST /api/auth/refresh", d.IAM.Refresh)
	mux.Handle("POST /api/auth/logout", d.Auth.Required(http.HandlerFunc(d.IAM.Logout)))
	mux.Handle("POST /api/auth/change-password", d.Auth.Required(http.HandlerFunc(d.IAM.ChangePassword)))
}

func registerMe(mux *http.ServeMux, d Deps) {
	mux.Handle("GET /api/me", d.Auth.Required(http.HandlerFunc(d.IAM.GetMe)))
	mux.Handle("PUT /api/me", d.Auth.Required(http.HandlerFunc(d.IAM.UpdateMe)))
	mux.Handle("PATCH /api/me", d.Auth.Required(http.HandlerFunc(d.IAM.UpdateMe)))
	mux.Handle("POST /api/me/password", d.Auth.Required(http.HandlerFunc(d.IAM.ChangePassword)))
	mux.Handle("POST /api/me/avatar", d.Auth.Required(http.HandlerFunc(d.IAM.Avatar)))
}

func registerDecks(mux *http.ServeMux, d Deps) {
	idem := func(method, pattern string, handler http.HandlerFunc) http.Handler {
		mw := middleware.Idempotent(middleware.IdempotencyOptions{
			Store: d.Idem, Scope: method + ":" + pattern,
		})
		return d.Auth.Required(mw(handler))
	}
	mux.Handle("GET /api/decks", d.Auth.Required(http.HandlerFunc(d.Decks.List)))
	mux.Handle("POST /api/decks", idem(http.MethodPost, "/api/decks", d.Decks.Create))
	mux.Handle("POST /api/decks/generate", idem(http.MethodPost, "/api/decks/generate", d.Decks.Generate))
	mux.Handle("GET /api/decks/{deckID}", d.Auth.Required(http.HandlerFunc(d.Decks.Get)))
	mux.Handle("PUT /api/decks/{deckID}", d.Auth.Required(http.HandlerFunc(d.Decks.Rename)))
	mux.Handle("PATCH /api/decks/{deckID}", d.Auth.Required(http.HandlerFunc(d.Decks.Rename)))
	mux.Handle("DELETE /api/decks/{deckID}", d.Auth.Required(http.HandlerFunc(d.Decks.Delete)))
	mux.Handle("POST /api/decks/{deckID}/cards",
		idem(http.MethodPost, "/api/decks/{deckID}/cards", d.Decks.AddCard))
	mux.Handle("PUT /api/decks/{deckID}/cards/{cardID}",
		d.Auth.Required(http.HandlerFunc(d.Decks.EditCard)))
	mux.Handle("PATCH /api/decks/{deckID}/cards/{cardID}",
		d.Auth.Required(http.HandlerFunc(d.Decks.EditCard)))
	mux.Handle("DELETE /api/decks/{deckID}/cards/{cardID}",
		d.Auth.Required(http.HandlerFunc(d.Decks.RemoveCard)))
	mux.Handle("POST /api/decks/{deckID}/reorder",
		d.Auth.Required(http.HandlerFunc(d.Decks.Reorder)))
	mux.Handle("PUT /api/decks/{deckID}/cards/order",
		d.Auth.Required(http.HandlerFunc(d.Decks.Reorder)))
	mux.Handle("GET /api/decks/{deckID}/progress",
		d.Auth.Required(http.HandlerFunc(d.Practice.Progress)))
}

func registerPractice(mux *http.ServeMux, d Deps) {
	idem := func(method, pattern string, handler http.HandlerFunc) http.Handler {
		mw := middleware.Idempotent(middleware.IdempotencyOptions{
			Store: d.Idem, Scope: method + ":" + pattern,
		})
		return d.Auth.Required(mw(handler))
	}
	mux.Handle("POST /api/practice/sessions",
		idem(http.MethodPost, "/api/practice/sessions", d.Practice.Start))
	mux.Handle("POST /api/practice/sessions/{sessionID}/ratings",
		d.Auth.Required(http.HandlerFunc(d.Practice.Rate)))
	mux.Handle("POST /api/practice/sessions/{sessionID}/finish",
		d.Auth.Required(http.HandlerFunc(d.Practice.Finish)))
	mux.Handle("GET /api/practice/sessions/{sessionID}/results",
		d.Auth.Required(http.HandlerFunc(d.Practice.Results)))
}
