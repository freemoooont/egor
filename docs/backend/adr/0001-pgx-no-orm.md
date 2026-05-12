# ADR 0001 — pgx/v5 + sqlc, no ORM

- **Status:** accepted
- **Date:** 2026-05-07
- **Deciders:** backend agent (TASK_ID `micocards-mvp`)

## Context

Micocards is a small Go service over PostgreSQL with a strict DDD layering. The data
layer needs to:

- Keep the domain layer free of any persistence concern (no struct tags, no implicit
  joins, no lazy loading).
- Give us first-class use of Postgres types (`uuid`, `timestamptz`, `smallint`,
  `jsonb` for the outbox payload, arrays for ordered card-id lists).
- Be readable and reviewable — the verifier and human reviewers must see the actual
  SQL, not derive it from method-call introspection.
- Stay testable end-to-end with `testcontainers-go` against a real Postgres.

Three families of options:

1. **An ORM** (gorm, ent, bun). Hides SQL behind reflection and runtime tag parsing.
2. **`database/sql` + a query builder** (squirrel, goqu).
3. **`pgx/v5` directly + sqlc-generated query code** from hand-written SQL files.

## Decision

**Option 3.** Use `github.com/jackc/pgx/v5` (with `pgxpool`) as the only Postgres
driver and `sqlc` to generate typed Go structs and methods from `*.sql` files.

Layout:

- Hand-written SQL in `backend/queries/<context>/*.sql`. One file per logical
  grouping (e.g. `users.sql`, `refresh_tokens.sql`).
- sqlc generates into `backend/internal/infrastructure/postgres/queries/<context>/`.
- Repository implementations live in
  `backend/internal/infrastructure/postgres/<context>repo/` and adapt the
  sqlc-generated types into the domain types defined in
  `backend/internal/domain/<context>/`.
- Migrations are managed by `goose` in `backend/migrations/<context>/` (separate
  schema per context — see ADR 0004).

## Alternatives considered

- **gorm.** Pulls reflection-heavy code into the domain via `gorm:"..."` tags or
  forces a separate persistence model anyway. Hard to tune queries, hard to audit
  what hits the DB. Rejected.
- **ent.** Code-first schema generation. Conflicts with our SQL-first migration
  story (goose) and with the "AI agents read raw SQL well" goal stated in
  `docs/stack.md`. Rejected.
- **`database/sql` + squirrel.** No type safety on returned columns; query strings
  built at runtime. Better than ORM but still strictly worse than sqlc for our
  pattern.

## Consequences

**Positive.**
- Domain structs stay 100% pure Go — no `db:"..."` or `json:"..."` tags.
- `git diff` on a query change shows the exact SQL.
- sqlc catches column-name drift at generate time, before tests run.
- pgx exposes `LISTEN/NOTIFY`, batch protocol, and `pgx.Tx` cleanly — the outbox
  pattern (ADR 0002) and the `UnitOfWork` port both lean on this.

**Negative.**
- Two-step workflow: edit SQL, run `make sqlc`, then write the repo adapter. Slower
  than ORM "just save the struct".
- No automatic migration generation from struct changes — but that is a feature; it
  forces explicit, append-only migrations (the spec requires this anyway).
- Generated code is committed-via-git (or, here, present in the working tree) so
  reviewers can read it.

**Neutral.**
- The `UnitOfWork` port wraps `pgx.Tx` and is passed via `context.Context` so
  repositories don't need to know whether they are inside a transaction.
- Integration tests use `testcontainers-go` to spin a real Postgres in CI/local.
