package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/micocards/api/internal/interfaces/http/dto"
)

// Recover is the panic guard. Any panic in a downstream handler is caught,
// logged with the request id, and translated to a 500 with the canonical
// envelope.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					rid := RequestIDFromContext(r.Context())
					log.ErrorContext(r.Context(), "panic recovered",
						slog.String("request_id", rid),
						slog.String("path", r.URL.Path),
						slog.String("stack", string(debug.Stack())),
						slog.Any("panic", rec),
					)
					writeJSON(w, http.StatusInternalServerError, dto.ErrorEnvelope{
						Error:   "internal_error",
						Message: fmt.Sprintf("internal server error: %v", rec),
						Details: map[string]any{"request_id": rid},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
