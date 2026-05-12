// Package middleware groups the cross-cutting HTTP middlewares: error mapping,
// recovery, request-id, slog access logging, JWT auth, and Idempotency-Key
// caching. None of them carry business logic — that lives in the use cases.
package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/interfaces/http/dto"
)

// ErrorMapping is one row in the table from ADR 0006.
type ErrorMapping struct {
	Sentinel error
	Status   int
	Code     string
}

// errorTable is the static map from sentinel → (status, code). Order matters
// only when two sentinels share a code, which we avoid by construction.
var errorTable = []ErrorMapping{
	// 401
	{iam.ErrUnauthorized, http.StatusUnauthorized, "unauthorized"},
	{iam.ErrInvalidCredentials, http.StatusUnauthorized, "invalid_credentials"},
	{iam.ErrRefreshTokenInvalid, http.StatusUnauthorized, "refresh_invalid"},
	{iam.ErrRefreshTokenExpired, http.StatusUnauthorized, "refresh_expired"},
	{iam.ErrRefreshTokenReused, http.StatusUnauthorized, "refresh_reused"},
	// 403
	{decks.ErrForbidden, http.StatusForbidden, "forbidden"},
	{practice.ErrForbidden, http.StatusForbidden, "forbidden"},
	// 404
	{iam.ErrUserNotFound, http.StatusNotFound, "user_not_found"},
	{decks.ErrDeckNotFound, http.StatusNotFound, "deck_not_found"},
	{decks.ErrCardNotFound, http.StatusNotFound, "card_not_found"},
	{practice.ErrSessionNotFound, http.StatusNotFound, "session_not_found"},
	// 409
	{iam.ErrEmailTaken, http.StatusConflict, "email_taken"},
	{iam.ErrIdempotencyConflict, http.StatusConflict, "idempotency_conflict"},
	{decks.ErrDeckDeleted, http.StatusConflict, "deck_deleted"},
	{practice.ErrSessionClosed, http.StatusConflict, "session_closed"},
	{practice.ErrSessionUntracked, http.StatusConflict, "session_untracked"},
	{practice.ErrSessionNotCompleted, http.StatusConflict, "session_not_completed"},
	// 422
	{iam.ErrInvalidEmail, http.StatusUnprocessableEntity, "invalid_email"},
	{iam.ErrInvalidDisplayName, http.StatusUnprocessableEntity, "invalid_display_name"},
	{iam.ErrInvalidPasswordHash, http.StatusUnprocessableEntity, "invalid_password_hash"},
	{iam.ErrPasswordTooWeak, http.StatusUnprocessableEntity, "password_too_weak"},
	{decks.ErrInvalidDeckTitle, http.StatusUnprocessableEntity, "invalid_deck_title"},
	{decks.ErrDeckTitleTooLong, http.StatusUnprocessableEntity, "deck_title_too_long"},
	{decks.ErrInvalidTerm, http.StatusUnprocessableEntity, "invalid_term"},
	{decks.ErrInvalidDefinition, http.StatusUnprocessableEntity, "invalid_definition"},
	{decks.ErrDeckCardLimitExceeded, http.StatusUnprocessableEntity, "deck_card_limit"},
	{decks.ErrInvalidCardReorder, http.StatusUnprocessableEntity, "invalid_card_reorder"},
	{decks.ErrDeckEmpty, http.StatusUnprocessableEntity, "deck_empty"},
	{practice.ErrCardNotInSession, http.StatusUnprocessableEntity, "card_not_in_session"},
	{practice.ErrInvalidRating, http.StatusUnprocessableEntity, "invalid_rating"},
	{practice.ErrInvalidPracticeMode, http.StatusUnprocessableEntity, "invalid_practice_mode"},
	{practice.ErrDeckEmpty, http.StatusUnprocessableEntity, "deck_empty"},
	// 400
	{iam.ErrIdempotencyKeyNeeded, http.StatusBadRequest, "idempotency_key_required"},
	// 501 / 502
	{decks.ErrAINotConfigured, http.StatusNotImplemented, "ai_not_configured"},
	{decks.ErrNotImplemented, http.StatusNotImplemented, "not_implemented"},
	{decks.ErrAIUpstream, http.StatusBadGateway, "ai_upstream"},
}

// ValidationError is the wire shape the validation adapter raises so the
// errorMapper can turn it into a 422 with field details.
type ValidationError struct {
	Code    string
	Message string
	Fields  map[string]string
}

// Error satisfies the error interface.
func (v *ValidationError) Error() string { return v.Message }

// NewValidationError builds a 422-bound error.
func NewValidationError(code, message string, fields map[string]string) *ValidationError {
	return &ValidationError{Code: code, Message: message, Fields: fields}
}

// JSONDecodeError is raised by the request-binding helpers when the body is
// malformed (syntactically invalid JSON, unknown fields, wrong types).
type JSONDecodeError struct {
	Message string
}

// Error satisfies the error interface.
func (e *JSONDecodeError) Error() string { return e.Message }

// WriteError serialises err into the canonical JSON envelope and the matching
// status code per ADR 0006.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	WriteErrorWithLogger(w, r, err, slog.Default())
}

// WriteErrorWithLogger is the testable variant.
func WriteErrorWithLogger(w http.ResponseWriter, r *http.Request, err error, log *slog.Logger) {
	if err == nil {
		return
	}
	// Validation errors first.
	var ve *ValidationError
	if errors.As(err, &ve) {
		body := dto.ErrorEnvelope{
			Error:   firstNonEmpty(ve.Code, "validation_failed"),
			Message: ve.Message,
		}
		if len(ve.Fields) > 0 {
			body.Details = map[string]any{"fields": ve.Fields}
		}
		writeJSON(w, http.StatusUnprocessableEntity, body)
		return
	}
	var jd *JSONDecodeError
	if errors.As(err, &jd) {
		writeJSON(w, http.StatusBadRequest, dto.ErrorEnvelope{
			Error:   "bad_request",
			Message: jd.Message,
		})
		return
	}
	for _, m := range errorTable {
		if errors.Is(err, m.Sentinel) {
			writeJSON(w, m.Status, dto.ErrorEnvelope{
				Error:   m.Code,
				Message: err.Error(),
			})
			return
		}
	}
	// Unknown error → 500. Log the full thing; reply with the request id only.
	rid := RequestIDFromContext(r.Context())
	if log != nil {
		log.ErrorContext(r.Context(), "internal error", slog.String("request_id", rid), slog.Any("err", err))
	}
	writeJSON(w, http.StatusInternalServerError, dto.ErrorEnvelope{
		Error:   "internal_error",
		Message: "internal server error",
		Details: map[string]any{"request_id": rid},
	})
}

// WriteJSON serialises v as JSON with the given status. Public so handlers can
// share the helper without importing internal packages.
func WriteJSON(w http.ResponseWriter, status int, v any) { writeJSON(w, status, v) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// MapStatus returns the status code that errorMapper would emit for err.
// Useful for tests and for the idempotency middleware (it must record the
// final response status).
func MapStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var ve *ValidationError
	if errors.As(err, &ve) {
		return http.StatusUnprocessableEntity
	}
	var jd *JSONDecodeError
	if errors.As(err, &jd) {
		return http.StatusBadRequest
	}
	for _, m := range errorTable {
		if errors.Is(err, m.Sentinel) {
			return m.Status
		}
	}
	return http.StatusInternalServerError
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
