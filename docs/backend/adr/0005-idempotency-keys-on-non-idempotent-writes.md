# ADR 0005 — Idempotency-Key on non-idempotent writes

- **Status:** accepted
- **Date:** 2026-05-07
- **Deciders:** backend agent (TASK_ID `micocards-mvp`)

## Context

Several use cases are non-idempotent by construction: `RegisterUser` mints a new
account and a new refresh-token family; `CreateDeck` creates a deck (and possibly
N cards); `AddCard` appends; `StartPracticeSession` creates a session;
`GenerateDeckWithAI` (when implemented) calls a paid upstream API.

A flaky network or an over-eager retry button on the client must not produce two
accounts, two decks, or two AI calls. Constraints from `docs/stack.md`:

- Stdlib `net/http`, no framework middleware library to lean on.
- Postgres-only persistence (no Redis or external dedup store).
- Spec line 152: "non-idempotent write endpoints accept an `Idempotency-Key`
  header".

## Decision

**Per-endpoint idempotency keys, persisted in `iam.idempotency_keys`, served via a
generic HTTP middleware.**

Mechanics:

1. Table `iam.idempotency_keys`:
   `scope text, key text, request_hash text, response_status int,
    response_headers jsonb, response_body bytea, expires_at timestamptz,
    PRIMARY KEY (scope, key)`.
2. The middleware (`internal/interfaces/http/middleware/idempotency.go`) wraps
   handlers tagged with `idempotent: true`. On request:
   a. If the request has no `Idempotency-Key` header and the route is tagged
      `idempotency-required`, return `400 Bad Request` (`ErrIdempotencyKeyRequired`).
   b. Otherwise compute `scope = "<METHOD>:<route-pattern>"` and look up
      `(scope, key)`.
   c. If found and `request_hash` matches the canonicalised request body+query, the
      cached response is replayed verbatim with header `Idempotent-Replay: true`.
   d. If found and `request_hash` differs, return `409 Conflict`
      (`ErrIdempotencyKeyConflict`).
   e. If not found, the handler runs; the middleware buffers the response and, on
      `2xx`, persists `(scope, key, request_hash, response, expires_at = now()+24h)`
      atomically with the use case's `pgx.Tx` via the `IdempotencyKeys` repository.
3. TTL: 24 hours. A janitor goroutine (the same one that vacuums the outbox; see
   ADR 0002) deletes expired rows.
4. Tagged endpoints (must accept the header):
   - `POST /api/auth/register`
   - `POST /api/decks`
   - `POST /api/decks/{deckID}/cards`
   - `POST /api/decks/generate`
   - `POST /api/practice/sessions`
   - `POST /api/me/avatar` (when implemented)

## Alternatives considered

- **No idempotency layer; rely on natural keys.** Works for some endpoints (e.g.
  `RateCard` is naturally idempotent on `(session_id, card_id)`) but not for
  `CreateDeck` or `RegisterUser`. Rejected as a global strategy.
- **Use `If-Match`/`If-None-Match` ETags.** Solves a different problem
  (optimistic concurrency on updates), not retry safety on creates.
- **Redis-backed idempotency.** Excluded by the Postgres-only constraint.

## Consequences

**Positive.**
- A retry storm produces exactly one side effect per `Idempotency-Key`.
- Replay returns the original response, so the client UI shows the same outcome.
- Implementation is one middleware + one repository, both <100 LOC.

**Negative.**
- Adds a Postgres write per non-idempotent request (small, hits a primary-key
  index).
- Clients must generate keys; we document the convention as
  `crypto.randomUUID()` in the frontend `apiFetch` wrapper.

**Neutral.**
- The `iam.idempotency_keys` table is in the `iam` schema even though it serves
  endpoints across all three contexts. This is intentional — it is a
  cross-cutting concern owned by the authentication context, which already runs
  in the request pipeline.
