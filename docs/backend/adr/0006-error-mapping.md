# ADR 0006 — Domain sentinel errors → HTTP status codes via central mapper

- **Status:** accepted
- **Date:** 2026-05-07
- **Deciders:** backend agent (TASK_ID `micocards-mvp`)

## Context

Domain code (`internal/domain/...`) emits typed sentinel errors
(`var ErrDeckNotFound = errors.New(...)`). Use cases propagate them. HTTP handlers
must turn them into the right status code with a stable, documented JSON shape so
the frontend (`apiFetch`) can branch on `error_code`.

We do not want every handler to ship its own `switch err {}`. We do not want the
domain to import `net/http`. We do not want the OpenAPI doc to drift from the actual
status codes.

## Decision

A single `errorMapper` lives in
`backend/internal/interfaces/http/middleware/errors.go`. It exposes:

```go
func WriteError(w http.ResponseWriter, r *http.Request, err error)
```

It walks the error chain with `errors.Is` against a static table and writes a JSON
response of the canonical shape:

```json
{
  "error":   "<machine code>",
  "message": "<human message, optional>",
  "details": { ... }
}
```

The mapping table is the single source of truth — both for the HTTP layer and for
the `responses.error.<code>` references in `docs/backend/openapi.yaml`.

### Mapping table

| Sentinel | HTTP status | `error` code |
|---|---|---|
| `ErrUnauthorized` | 401 | `unauthorized` |
| `ErrInvalidCredentials` | 401 | `invalid_credentials` |
| `ErrRefreshTokenInvalid` | 401 | `refresh_invalid` |
| `ErrRefreshTokenExpired` | 401 | `refresh_expired` |
| `ErrRefreshTokenReused` | 401 | `refresh_reused` |
| `ErrForbidden` | 403 | `forbidden` |
| `ErrUserNotFound` | 404 | `user_not_found` |
| `ErrDeckNotFound` | 404 | `deck_not_found` |
| `ErrCardNotFound` | 404 | `card_not_found` |
| `ErrSessionNotFound` | 404 | `session_not_found` |
| `ErrEmailTaken` | 409 | `email_taken` |
| `ErrIdempotencyKeyConflict` | 409 | `idempotency_conflict` |
| `ErrDeckDeleted` | 409 | `deck_deleted` |
| `ErrSessionClosed` | 409 | `session_closed` |
| `ErrSessionUntracked` | 409 | `session_untracked` |
| `ErrSessionNotCompleted` | 409 | `session_not_completed` |
| `ErrInvalidEmail` | 422 | `invalid_email` |
| `ErrInvalidDisplayName` | 422 | `invalid_display_name` |
| `ErrInvalidPasswordHash` | 422 | `invalid_password_hash` |
| `ErrPasswordTooWeak` | 422 | `password_too_weak` |
| `ErrInvalidDeckTitle` | 422 | `invalid_deck_title` |
| `ErrDeckTitleTooLong` | 422 | `deck_title_too_long` |
| `ErrInvalidTerm` | 422 | `invalid_term` |
| `ErrInvalidDefinition` | 422 | `invalid_definition` |
| `ErrDeckCardLimitExceeded` | 422 | `deck_card_limit` |
| `ErrInvalidCardReorder` | 422 | `invalid_card_reorder` |
| `ErrCardNotInSession` | 422 | `card_not_in_session` |
| `ErrInvalidRating` | 422 | `invalid_rating` |
| `ErrInvalidPracticeMode` | 422 | `invalid_practice_mode` |
| `ErrDeckEmpty` | 422 | `deck_empty` |
| `ErrIdempotencyKeyRequired` | 400 | `idempotency_key_required` |
| `ErrAIUpstream` | 502 | `ai_upstream` |
| `ErrNotImplemented` | 501 | `not_implemented` |

Anything else (panic, unmapped error) becomes `500 internal_error` with the original
err logged via `slog.ErrorContext`. The recovery middleware sets a request-id header
so the user can quote it.

## Alternatives considered

- **Per-handler switch.** Duplicate code; drift between handlers; impossible to keep
  in sync with OpenAPI. Rejected.
- **Tagged errors via custom interface.** `type httpStatus interface { Status() int }`
  on every domain error. Couples domain to HTTP semantics. Rejected.
- **gRPC-style codes everywhere then map at the edge.** Overkill; we are REST-only
  per spec.

## Consequences

**Positive.**
- One file (`errors.go`) is the contract. Adding a sentinel is one row added.
- The same table powers the `errors` array in OpenAPI generation and the
  Reatom-side `apiFetch` discriminator.
- Tests: a single table-driven test asserts every sentinel maps as documented.

**Negative.**
- New error sentinels must be registered in two places: the `errors.go` file and
  the OpenAPI doc. The verifier's lint script catches drift.

**Neutral.**
- Validation errors from `go-playground/validator` are translated into
  `ErrInvalidEmail`/`ErrInvalidDeckTitle`/etc. by a small adapter in the HTTP layer
  (`dto.bindAndValidate`). Domain code never sees `validator.ValidationErrors`.
