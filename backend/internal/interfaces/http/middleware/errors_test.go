package middleware_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/interfaces/http/middleware"
)

func TestErrorMapper_KnownSentinels(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{iam.ErrUnauthorized, 401, "unauthorized"},
		{iam.ErrInvalidCredentials, 401, "invalid_credentials"},
		{iam.ErrRefreshTokenInvalid, 401, "refresh_invalid"},
		{iam.ErrRefreshTokenExpired, 401, "refresh_expired"},
		{iam.ErrRefreshTokenReused, 401, "refresh_reused"},
		{iam.ErrEmailTaken, 409, "email_taken"},
		{iam.ErrIdempotencyConflict, 409, "idempotency_conflict"},
		{iam.ErrIdempotencyKeyNeeded, 400, "idempotency_key_required"},
		{iam.ErrInvalidEmail, 422, "invalid_email"},
		{iam.ErrPasswordTooWeak, 422, "password_too_weak"},
		{iam.ErrUserNotFound, 404, "user_not_found"},
		{decks.ErrDeckNotFound, 404, "deck_not_found"},
		{decks.ErrCardNotFound, 404, "card_not_found"},
		{decks.ErrForbidden, 403, "forbidden"},
		{decks.ErrAINotConfigured, 501, "ai_not_configured"},
		{decks.ErrAIUpstream, 502, "ai_upstream"},
		{decks.ErrInvalidDeckTitle, 422, "invalid_deck_title"},
		{decks.ErrDeckCardLimitExceeded, 422, "deck_card_limit"},
		{practice.ErrSessionNotFound, 404, "session_not_found"},
		{practice.ErrSessionClosed, 409, "session_closed"},
		{practice.ErrSessionUntracked, 409, "session_untracked"},
		{practice.ErrSessionNotCompleted, 409, "session_not_completed"},
		{practice.ErrInvalidRating, 422, "invalid_rating"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/x", nil)
			middleware.WriteError(rr, req, c.err)
			require.Equal(t, c.status, rr.Code)
			var body map[string]any
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			require.Equal(t, c.code, body["error"])
			require.Equal(t, c.status, middleware.MapStatus(c.err))
		})
	}
}

func TestErrorMapper_WrappedSentinel(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	wrapped := fmt.Errorf("wrap: %w", iam.ErrInvalidCredentials)
	middleware.WriteError(rr, req, wrapped)
	require.Equal(t, 401, rr.Code)
}

func TestErrorMapper_ValidationError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	middleware.WriteError(rr, req, middleware.NewValidationError("validation_failed", "bad", map[string]string{"email": "required"}))
	require.Equal(t, 422, rr.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, "validation_failed", body["error"])
	require.NotNil(t, body["details"])
}

func TestErrorMapper_JSONDecodeError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	middleware.WriteError(rr, req, &middleware.JSONDecodeError{Message: "boom"})
	require.Equal(t, 400, rr.Code)
}

func TestErrorMapper_UnknownError(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	middleware.WriteError(rr, req, errors.New("mystery"))
	require.Equal(t, 500, rr.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, "internal_error", body["error"])
}

func TestMapStatus_Nil(t *testing.T) {
	require.Equal(t, 200, middleware.MapStatus(nil))
}

func TestMapStatus_ValidationAndDecodeAndUnknown(t *testing.T) {
	require.Equal(t, 422, middleware.MapStatus(middleware.NewValidationError("c", "m", nil)))
	require.Equal(t, 400, middleware.MapStatus(&middleware.JSONDecodeError{Message: "m"}))
	require.Equal(t, 500, middleware.MapStatus(errors.New("x")))
}
