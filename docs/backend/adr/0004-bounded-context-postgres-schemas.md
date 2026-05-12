# ADR 0004 — One Postgres schema per bounded context

- **Status:** accepted
- **Date:** 2026-05-07
- **Deciders:** backend agent (TASK_ID `micocards-mvp`)

## Context

DDD requires us to enforce context boundaries at the data layer. The naive shared-DB
"one schema, prefix by context name" approach drifts almost immediately: someone adds
a join from `decks_decks` to `iam_users` "just for the dashboard", and the
boundary is dead. The spec restricts us to Postgres only, but does not restrict us to
a single schema.

## Decision

Each bounded context gets its **own Postgres schema** in the same database:

- `iam` — `iam.users`, `iam.refresh_tokens`, `iam.idempotency_keys`, `iam.outbox`.
- `decks` — `decks.decks`, `decks.cards`, `decks.outbox`.
- `practice` — `practice.sessions`, `practice.rated_cards`,
  `practice.user_deck_progress`, `practice.outbox`.

Rules:

1. **No cross-schema joins.** Migrations and queries that reference
   `<context>.<table>` are restricted to that context's own schema, plus the
   `public` schema for shared extensions (`pgcrypto`, `uuid-ossp`).
2. **Foreign keys do not cross schemas.** A `decks.decks.owner_id` is a `uuid`
   without a database FK to `iam.users.id`. Referential integrity is enforced by
   the application layer (the JWT carries the user id; the use case validates
   ownership).
3. **One sqlc config per schema.** `backend/sqlc.yaml` declares three packages,
   one per schema, generated into
   `backend/internal/infrastructure/postgres/queries/<context>/`.
4. **Migrations per context.** `backend/migrations/<context>/` holds goose-managed
   migrations. The migration runner applies all three subfolders in a fixed order
   (`iam → decks → practice`) on `make migrate-up`.
5. **Search path.** Each repository sets `SET LOCAL search_path TO <context>` at the
   start of its tx via the `UnitOfWork` port. Cross-context accidents fail at parse
   time.
6. **Read models cross schemas only via published events.** `UserDeckProgress` lives
   in `practice` and is fed by `PracticeSessionCompleted` events from itself plus
   `DeckDeleted`/`CardRemoved` events from `decks`. It never `JOIN`s.

## Alternatives considered

- **One schema, table-name prefixes.** Loses linter-level enforcement; a `JOIN
  decks_decks d JOIN iam_users u` compiles fine. Rejected.
- **One database per context.** Stronger isolation but blows up local dev (three
  containers, three connection strings, three migration runs). Overkill for v1.
- **Postgres logical replication / multiple databases on one cluster.** Same cost as
  above without the isolation gain.

## Consequences

**Positive.**
- Mistakes that cross the boundary are loud, not silent.
- Clean migration story: each context evolves its schema independently.
- Splitting into separate services later is a `pg_dump --schema=<context>` away.

**Negative.**
- No DB-level FK between, e.g., `decks.decks.owner_id` and `iam.users.id`. We pay
  for this with explicit ownership checks in the application layer.
- Slightly more `search_path` ceremony at the connection layer.

**Neutral.**
- `pg_dump`/`pg_restore` operate per-schema, simplifying backups (`pg_dump
  --schema=iam` for the auth-only export).
