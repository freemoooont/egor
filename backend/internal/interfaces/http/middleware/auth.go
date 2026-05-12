package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/micocards/api/internal/domain/iam"
)

// AccessTokenVerifier is the small port the auth middleware needs. The infra
// JWT signer satisfies it (its VerifyAccessToken returns iam.ErrUnauthorized
// on any failure).
type AccessTokenVerifier interface {
	VerifyAccessToken(ctx context.Context, token string) (userID string, err error)
}

// userIDKey is the context key carrying the authenticated user id.
type userIDKey struct{}

// UserIDFromContext returns the authenticated user id, or "" if the request
// reached this point without going through Auth.Required.
func UserIDFromContext(ctx context.Context) string {
	v := ctx.Value(userIDKey{})
	if v == nil {
		return ""
	}
	id, _ := v.(string)
	return id
}

// WithUserID attaches the authenticated user id to ctx (exposed for tests).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// Auth carries the JWT verifier and exposes both required and optional
// middleware constructors.
type Auth struct {
	Verifier AccessTokenVerifier
}

// NewAuth builds the middleware bundle.
func NewAuth(v AccessTokenVerifier) *Auth { return &Auth{Verifier: v} }

// Required parses Authorization: Bearer <token> and rejects with 401 on any
// failure. Successful requests carry the authenticated user id on ctx.
func (a *Auth) Required(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerFromHeader(r.Header.Get("Authorization"))
		if !ok {
			WriteError(w, r, iam.ErrUnauthorized)
			return
		}
		userID, err := a.Verifier.VerifyAccessToken(r.Context(), token)
		if err != nil || userID == "" {
			WriteError(w, r, iam.ErrUnauthorized)
			return
		}
		ctx := WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Optional decorates the request with the user id when a valid token is
// present, but never rejects. Used by the AI generation handler.
func (a *Auth) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerFromHeader(r.Header.Get("Authorization"))
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		userID, err := a.Verifier.VerifyAccessToken(r.Context(), token)
		if err == nil && userID != "" {
			r = r.WithContext(WithUserID(r.Context(), userID))
		}
		next.ServeHTTP(w, r)
	})
}

func bearerFromHeader(h string) (string, bool) {
	const prefix = "Bearer "
	if len(h) < len(prefix) {
		return "", false
	}
	if !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	tok := strings.TrimSpace(h[len(prefix):])
	if tok == "" {
		return "", false
	}
	return tok, true
}
