# Backend Claude guide — Micocards

See the root `CLAUDE.md` first for cross-cutting rules (orchestrator-only,
TDD discipline, DDD-docs-first). This file is the **authoritative** backend stack
guide; when you are inside `backend/`, this file wins over the root for backend
specifics.

## Layout

```
backend/
├── cmd/
│   └── api/main.go                       # composition root
├── internal/
│   ├── domain/{iam,decks,practice}/      # pure domain — imports nothing project-internal
│   ├── application/{iam,decks,practice}/ # use cases; imports only domain
│   ├── infrastructure/
│   │   ├── postgres/
│   │   │   ├── {iamrepo,decksrepo,practicerepo}/
│   │   │   └── queries/{iam,decks,practice}/   # sqlc-generated
│   │   ├── auth/                         # bcrypt, JWT signer
│   │   └── ai/                           # AI stub (returns 501 unless AI_API_KEY set)
│   └── interfaces/
│       └── http/
│           ├── router.go                 # http.ServeMux + middleware chain
│           ├── middleware/{auth,idempotency,errors,recover,requestid}.go
│           ├── {iam,decks,practice}/     # handlers per context
│           └── dto/                      # request/response DTOs
├── migrations/{iam,decks,practice}/      # goose, append-only
├── queries/{iam,decks,practice}/         # sqlc input
├── sqlc.yaml
├── .golangci.yaml
├── go.mod
└── Makefile
```

## Dependency rule (MUST)

`interfaces → application → domain` and `infrastructure → application/domain`. Domain
imports nothing from the project. Application imports only domain. Infrastructure and
interfaces may import both. Repositories are **interfaces** in `domain/<context>/`,
implementations in `infrastructure/postgres/<context>repo/`. Use cases open all
transactions via the `UnitOfWork` port; repositories accept the tx through `context.Context`,
not as a global.

DTOs at the boundary, never inside the domain. Map domain ↔ DTO at the edge. Never
expose `*domain.Deck` over JSON.

## TDD discipline

Domain and use-case layers are written **test-first** with `github.com/stretchr/testify`.
Test names match the invariant names in `docs/backend/aggregates.md` (e.g.
`TestDeck_TitleLengthBetween1And120`). Coverage gates: `≥ 80%` on `internal/domain/...`
and `internal/application/...`. Verify with:

```
go test -coverprofile=cov.out ./internal/domain/... && go tool cover -func=cov.out | tail -1
go test -coverprofile=cov.out ./internal/application/... && go tool cover -func=cov.out | tail -1
```

Infrastructure tests use `testcontainers-go` with a real Postgres. At least one
`*_integration_test.go` per context lives under
`internal/infrastructure/postgres/<context>repo/`. Tests build their own schema by
running the goose migrations against the container.

The DDD docs in `docs/backend/` are **frozen before any Go code is written**. Every
glossary term in `docs/backend/ubiquitous-language.md` is referenced from
`aggregates.md` or `use-cases.md`; every aggregate has at least one numbered invariant;
every use case lists Input, Output, Invariants, Side effects; every event lists a
publisher use case. `scripts/ddd-lint.sh` (root) enforces this.

## pgx/v5 + sqlc + goose conventions

- Driver: `github.com/jackc/pgx/v5` with `pgxpool`. No `database/sql`. No ORM.
- sqlc: queries in `queries/<context>/*.sql`, generated into
  `internal/infrastructure/postgres/queries/<context>/`. Run `make sqlc` after every
  query edit; do not hand-edit generated code.
- goose: migrations in `migrations/<context>/<unix-ts>_<name>.up.sql` /
  `<unix-ts>_<name>.down.sql`. **Append-only — never edit a merged migration**; write
  a new one. Each context applies its migrations to its own schema (`iam`, `decks`,
  `practice`); `make migrate-up` runs all three subfolders in order.
- One `pgx.Tx` per use case, opened by the `UnitOfWork` port. Repositories pull the
  active tx out of `context.Context` — they never receive a connection directly.
- Times: UTC, persisted as `timestamptz`. Money/scoring: integers only — never floats.

## HTTP conventions

- Stdlib `net/http` + `http.ServeMux` with method+path patterns
  (`mux.HandleFunc("POST /api/decks", ...)`). No chi/gin/echo.
- Auth: `Authorization: Bearer <jwt>`. Middleware in
  `internal/interfaces/http/middleware/auth.go` rejects unauthenticated calls to
  protected routes with `401`. Public routes: `/api/healthz`, `/api/auth/register`,
  `/api/auth/login`, `/api/auth/refresh`.
- Idempotency middleware honours the `Idempotency-Key` header on tagged endpoints
  (see ADR 0005). Replays return the cached response with `Idempotent-Replay: true`.
- Error mapping: every handler returns either a DTO or an error; the central
  `errorMapper` in `middleware/errors.go` turns sentinels into status codes per
  ADR 0006. The mapping table is the source of truth — keep `docs/backend/openapi.yaml`
  in sync.
- Validation: `go-playground/validator/v10` on DTOs. Translate validation errors into
  domain `ErrInvalid*` sentinels in `dto.bindAndValidate`.
- Logging: `slog` JSON in prod, text in dev. Every request gets a request-id via
  `middleware/requestid.go`; pass `context.Context` everywhere.

## Run commands

- `make lint` — `golangci-lint run ./...` (config: `.golangci.yaml`; enables
  `errcheck`, `govet`, `staticcheck`, `revive`, `gocritic`, `gosec`, `sqlclosecheck`).
- `make backend-test` — `go test ./... -count=1` (unit only, default).
- `make backend-test-integration` — `INTEGRATION=1 go test -tags=integration
  ./... -count=1`. Requires Docker (testcontainers spins up Postgres 16).
- `make backend-sqlc` — regenerate `internal/infrastructure/postgres/queries/`.
  When sqlc is not on PATH the target prints a skip notice; `make backend-tools`
  installs it via `go install`.
- `make backend-migrate-up` / `make backend-migrate-down` — runs goose against
  `$DATABASE_URL` across `migrations/{shared,iam,decks,practice}` in order.
  When goose is missing, integration tests still apply migrations through the
  in-process `internal/infrastructure/postgres/migrate` package.
- `make backend-tools` — best-effort `go install` of sqlc and goose binaries.
- `make postgres-up` / `make postgres-down` — bring the local
  `infra/docker-compose.yaml` Postgres up on `127.0.0.1:55432`.
- `make backend-run-api` — plain `go run ./cmd/api`.

### Test build tags

- Unit tests build with the default tag set and never touch Docker.
- Integration tests are guarded by `//go:build integration` and the
  `testdb.New` helper short-circuits when `INTEGRATION=0`. Run them via
  `make backend-test-integration` or `INTEGRATION=1 go test -tags=integration ./...`.

## No business logic in shared packages

`internal/infrastructure/auth/`, `internal/interfaces/http/middleware/`,
`internal/infrastructure/ai/` are **adapters**. Business rules live in
`internal/domain/<context>/`. If you find yourself encoding "if user has more than X
decks, …" in a middleware, move it.

## Context propagation

Every function below the HTTP layer takes `context.Context` as its first argument —
including repository methods and event subscribers. Pass the request's context
through the use case, the unit of work, and into pgx. Never use
`context.Background()` outside `cmd/api/main.go`.

