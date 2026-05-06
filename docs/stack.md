# Stack

## Frontend

### Core

- **Vite+** — toolchain (Vite, Rolldown, Vitest, tsdown, Oxlint, Oxfmt). Use the `vp` CLI: `vp dev`, `vp build`, `vp check`, `vp test`, `vp lint`, `vp install`. **Never call vite/vitest/oxlint directly. Never use pnpm/npm/yarn directly** — use `vp add`, `vp remove`. Run `vp run <script>` for custom package.json scripts whose name collides with a built-in.
- **React 19** + **TypeScript 6**.
- **Reatom v1000** (`@reatom/core`, `@reatom/react`) — state, routing (`reatomRoute`), forms (`reatomField`/`reatomForm`), async (`computed(...).extend(withAsyncData())` for reads, `action(...).extend(withAsync())` for mutations). Components are written as `reatomComponent`, not `React.memo` + `useState`.
- **Zod 4** — schema validation, integrated with Reatom forms and route params.
- **MSW** — API mocking in development.

### UI layer (chosen: Tailwind + shadcn/ui)

- **Tailwind CSS v4** — utility-first styling. Tokens (colors, radii, spacing) live in `tailwind.config.ts` so the orange Micocards brand and other Figma tokens map to theme values, not hardcoded hexes.
- **shadcn/ui** — copy-paste components built on Radix UI primitives. Components are generated into `src/shared/ui/` via `vp dlx shadcn@latest add <component>` and then **owned by this repo** — agents edit the source directly to match Figma. Provides accessibility (Radix), full customization, and high familiarity for LLM agents.
- **lucide-react** — icon set used by shadcn/ui by default.
- **No Mantine.** Do not install `@mantine/*`.

### PWA

- **vite-plugin-pwa** with `registerType: 'autoUpdate'` and Workbox runtime caching. Manifest (name, icons, theme color matching the brand orange, `display: 'standalone'`) lives in `vite.config.ts`. Icons + `apple-touch-icon` go in `public/`. App must be installable and work offline for already-visited deck/practice screens.

### Architecture: Feature-Sliced Design (FSD v2.1)

Path alias: `@/` → `src/` (configured in `vite.config.ts`).

```
src/
├── app/        # shell, providers, routes registry, global styles (segments only, no slices)
├── pages/      # one slice per page; index.ts re-exports route
├── widgets/    # composite UI blocks reused on 2+ pages (layouts, shells)
├── features/   # reusable user interactions (login, create-deck, rate-card)
├── entities/   # reusable business domain models (user, deck, card)
└── shared/     # infrastructure: ui (shadcn), api client, router root, config, lib, auth (segments only, no slices)
```

#### Layer rules (MUST)

- **Import direction (top-down only):** `app → pages → widgets → features → entities → shared`. Upward imports and cross-imports between slices on the same layer are forbidden.
- **Public API:** every slice exposes its surface through `index.ts`. External consumers import only from there — never deep-import internal files.
  - ✅ `import { LoginForm } from '@/features/auth'`
  - ❌ `import { LoginForm } from '@/features/auth/ui/LoginForm'`
- **Domain-based file names**, not technical-role names. Use `model/deck.ts`, `api/fetch-deck.ts`. **Never** `types.ts`, `utils.ts`, `helpers.ts` — they mix unrelated domains in one file.
- **No business logic in `shared/`.** Shared is infrastructure: UI kit, utilities, API client, route constants, auth tokens, env config, assets. Domain rules go in `entities/` or higher.
- **No cross-imports between slices on the same layer.** Resolve via the order in §"Cross-import resolution" below.

#### Pages First (the v2.1 default)

Place new code in `pages/<slice>/` first. Duplication across pages is acceptable — extract to a lower layer **only when 2+ consumers exist and the team agrees**. An entity or feature used by only one page should stay in that page (Steiger: `insignificant-slice`).

Decision tree:

1. Used in only one page? → keep in `pages/<slice>/`.
2. Reusable infrastructure with no business logic? → `shared/{ui,lib,api,auth,config}/`.
3. Complete user action reused in 2+ places? → `features/`.
4. Business domain model reused in 2+ places? → `entities/`.
5. App-wide config/providers/router? → `app/`.

When in doubt, keep it in `pages/`.

#### Standard segments inside a slice

- `ui/` — components, styles
- `model/` — data models, atoms/state, business logic, validation
- `api/` — backend calls, request functions, API-specific types
- `lib/` — slice-internal utilities
- `config/` — slice-local config, feature flags

In `app/` and `shared/`, code is organized by segments directly (no slices); segments within these layers may import from each other.

#### Cross-import resolution (in order)

1. **Merge slices** — if they always change together they're one slice.
2. **Extract shared logic to `entities/`** — keep UI in features/widgets.
3. **Compose in a higher layer (IoC)** — pages/app import both and wire them via props/slots.
4. **`@x` notation** — last resort, between entities only; document why steps 1–3 don't apply.

#### Anti-patterns to avoid

- Premature `entities/` — don't create one for a model used by a single page.
- CRUD in `entities/` — CRUD belongs in `shared/api/`.
- A `user` entity created just for auth tokens — put auth in `shared/auth/`.
- Single-use features/entities — keep them in their page.
- "God slices" with broad responsibilities — split into focused slices.
- Adding `ui/` to entities and importing it from another entity — never cross-import entity UI; only consume from features/widgets/pages.

Validate with the official linter when in doubt: `vp dlx @feature-sliced/steiger src`.

### Initialization order (critical, mirrors reference)

1. `src/setup.ts` runs `clearStack()` and creates `rootFrame = context.start()`. **Imported first.**
2. `src/main.tsx` wraps `<App />` in `<reatomContext.Provider value={rootFrame}>`.
3. `app/routes.ts` re-exports every page route — required to register them with the router.
4. `rootRoute.render()` inside `<App />` renders the matched route.

### Reatom: use the MOST advanced patterns (mandatory)

The bar is not "working Reatom code" — it is **top-tier, production-grade Reatom**. Before writing any atom, action, computed, route, or form, read `llms/reatom.md` end-to-end and pick the most advanced applicable pattern. Default to the heavy-machinery API, not the toy examples.

Specifically:

- **Read the full `llms/reatom.md` every session.** Do not rely on memory — the v1000 API evolves and many "obvious" idioms are anti-patterns.
- **Async reads → `computed(async ...).extend(withAsyncData())`** with proper cache, retry, and abort handling. Never `useEffect` + `fetch`. Never imperative loaders.
- **Mutations → `action(...).extend(withAsync())`** with `onFulfill` / `onReject` handlers and proper cancellation; chain dependent state through actions, not effects.
- **Memoize expensive computeds** and split big atoms into focused ones — fine-grained reactivity is the whole point. One atom per concern; derive everything else with `computed`.
- **Forms**: `reatomForm` + `reatomField` with Zod schemas — wire validation, async submit, error mapping, dirty/touched state through Reatom, not local React state.
- **Routing**: `reatomRoute` with typed params (Zod), `loader` for data, nested routes for master-detail. Use `self.outlet()` correctly; don't render children manually.
- **Performance hygiene**: avoid creating atoms inside components, avoid recreating actions per render (always module-scope or `reatomComponent` factory), use `reatomComponent` for every component to get auto-tracked re-renders, prefer derived `computed` over recalculation.
- **No identity actions**, no double-state (atom + local `useState`), no manual subscriptions, no `effect` for fetching, no untracked side-channels around the reactive frame.
- **Always name** every atom/action/computed: `atom(0, 'feature.name')`. Unnamed primitives are invisible in devtools and hurt debuggability.

If a pattern in this codebase looks simpler than what `llms/reatom.md` shows for the same problem, it is probably wrong — escalate to the advanced API.

### CRITICAL: Reatom `wrap` rule

`setup.ts` calls `clearStack()`, so atom mutations outside the reactive frame fail silently with `ReatomError: missing async stack`. **Wrap every async boundary** that touches atoms or actions:

- DOM handlers: `onClick={wrap(() => atom.set(...))}`
- `.then()` callbacks, `addEventListener`, `setTimeout`.
- Curried handlers: wrap the inner — `(id) => wrap(() => {...})`, not `wrap((id) => () => {...})`.
- Don't chain after `wrap` — wrap each step.

Full Reatom v1000 API + patterns + anti-patterns: read `llms/reatom.md` (copy from reference repo) **before any Reatom work**. Do not rely on memory.

### Reference project

`/Users/vladislav.molotsilo/WebstormProjects/x5/kot` — same Vite+/Reatom/React/FSD foundation. Use it as the source of truth for:

- `vite.config.ts` shape (proxy, lint, fmt, alias).
- FSD layout, `setup.ts`, `main.tsx`, `app/routes.ts`, `shared/router.ts` patterns.
- Reatom routing: root route, page route, nested master-detail, loaders.
- `shared/api/` client (`apiFetch` with `wrap`).
- `llms/reatom.md` — copy verbatim, it is the canonical Reatom v1000 doc.
- `AGENTS.md` and `CLAUDE.md` — read both; the wrap rule and `vp` workflow rules apply identically here.

**Differences from reference:** no Mantine → use Tailwind + shadcn/ui; add `vite-plugin-pwa`; no `@x5/*` packages.

### Conventions

- Always name atoms/actions/computed: `atom(0, 'feature.name')`.
- Data fetching → `computed(async () => ...).extend(withAsyncData())`. Mutations → `action(...).extend(withAsync())`. Never use `effect` or imperative fetch for either.
- No identity actions (actions that just forward into an atom — call `atom.set` directly).
- Always run `vp check` (format + lint + typecheck) and `vp test` before claiming a task done.

---

## Backend

### Core

- **Go** (latest stable, `go.mod` toolchain pinned). Use `go 1.22+` HTTP routing in stdlib `net/http` (`http.ServeMux` with method+path patterns) — no need for chi/gin unless a concrete need appears.
- **PostgreSQL** as the only persistence store. No Redis/Mongo/etc. unless explicitly added later.
- **pgx/v5** (`github.com/jackc/pgx/v5` + `pgxpool`) — direct driver, not `database/sql`. Better Postgres type support, prepared statements, batching, `LISTEN/NOTIFY`.
- **sqlc** — generate typed Go code from SQL files. Queries live in `*.sql`, generated structs/methods in `internal/infrastructure/postgres/queries/`. AI agents read raw SQL well; this keeps the data layer auditable.
- **goose** — SQL migrations in `migrations/` (timestamp-prefixed, `*.up.sql` / `*.down.sql`). Run via `goose -dir migrations postgres "$DATABASE_URL" up`.
- **slog** (stdlib `log/slog`) — structured logging, JSON handler in prod, text handler in dev.
- **go-playground/validator/v10** — struct-tag validation for HTTP DTOs at the boundary.
- **golang-jwt/jwt/v5** — JWT auth (access + refresh tokens). Tokens issued by an auth use case, verified in HTTP middleware.
- **bcrypt** (`golang.org/x/crypto/bcrypt`) — password hashing.
- **testify** + **testcontainers-go** (Postgres container) for integration tests against a real DB. Unit tests use stdlib `testing` only.
- **air** (dev only) — live reload during local development.

### CRITICAL: brainstorm-driven architecture before any code

**Before writing a single line of backend code**, run the `superpowers:brainstorming` skill to produce the architectural documents for DDD. No coding starts until these docs are frozen and reviewed. This is non-negotiable — DDD without ubiquitous language and explicit boundaries collapses into anemic CRUD.

Required output of the brainstorm phase, stored in `docs/backend/`:

1. **`ubiquitous-language.md`** — glossary of domain terms (Deck, Card, Practice Session, Rating, User, Progress) with definitions agreed across product/design/code. Names in code MUST match this glossary verbatim — no synonyms.
2. **`bounded-contexts.md`** — list of bounded contexts (likely: `iam` for auth/users, `decks` for deck+card management, `practice` for practice sessions and progress tracking). Each context owns its own aggregates, repositories, and DB schema (separate Postgres schemas or table prefixes).
3. **`aggregates.md`** — for each aggregate: root entity, invariants, transactional boundary, identifying events. Example: `Deck` is an aggregate root; `Card` is part of the `Deck` aggregate (no card mutation outside its deck).
4. **`domain-events.md`** — events emitted by the domain (`DeckCreated`, `CardRated`, `PracticeSessionCompleted`). Even if delivery starts in-process, define the contract now.
5. **`use-cases.md`** — application services / use cases per context (e.g. `CreateDeck`, `GenerateDeckWithAI`, `StartPracticeSession`, `RateCard`, `FinishPracticeSession`). Each entry: input DTO, output DTO, invariants enforced, side effects.
6. **`adr/`** — `adr/0001-*.md`, `adr/0002-*.md` ... Architecture Decision Records for the load-bearing choices (why pgx+sqlc, why no ORM, transactional outbox vs synchronous events, auth strategy, etc.).

If any of these is missing, the implementation is blocked — go back to brainstorming.

### Architecture: Domain-Driven Design (layered, dependency points inward)

```
backend/
├── cmd/
│   └── api/main.go                    # composition root: wire deps, start http server
├── internal/
│   ├── domain/                        # PURE domain — no imports from infra/app/transport
│   │   ├── deck/                      # one package per bounded context
│   │   │   ├── deck.go                # aggregate root + entity types
│   │   │   ├── card.go                # entity inside Deck aggregate
│   │   │   ├── repository.go          # repository INTERFACE (defined here, implemented in infra)
│   │   │   ├── events.go              # domain events
│   │   │   └── errors.go              # domain-specific sentinel errors
│   │   ├── practice/
│   │   └── iam/
│   ├── application/                   # use cases, orchestration, transactions
│   │   ├── deck/
│   │   │   ├── create_deck.go         # one file per use case
│   │   │   ├── generate_with_ai.go
│   │   │   └── ports.go               # outbound ports (interfaces) the UC needs
│   │   └── practice/
│   ├── infrastructure/                # adapters: DB, external APIs, clock, id-gen
│   │   ├── postgres/
│   │   │   ├── deckrepo/              # implements domain/deck.Repository
│   │   │   ├── practicerepo/
│   │   │   └── queries/               # sqlc-generated code
│   │   ├── ai/                        # OpenAI/etc. client for AI deck generation
│   │   └── auth/                      # bcrypt, jwt
│   └── interfaces/                    # transport adapters
│       └── http/
│           ├── router.go              # mux + middleware
│           ├── deck/                  # handlers per context
│           ├── middleware/            # auth, logging, recovery, request id
│           └── dto/                   # request/response DTOs (NOT domain types)
├── migrations/                        # goose SQL migrations
├── queries/                           # sqlc input .sql files (one folder per context)
├── sqlc.yaml
├── go.mod
└── docs/backend/                      # brainstorm artifacts (see above)
```

#### Layer rules (MUST)

- **Dependency rule**: `interfaces → application → domain` and `infrastructure → application/domain`. The domain layer imports nothing from the project. The application layer imports only the domain. Interfaces and infrastructure may import application + domain. Never the other way around.
- **Repositories are interfaces in `domain/<context>/`**, implemented in `infrastructure/postgres/<context>repo/`. The application layer depends on the interface; main.go wires the concrete implementation.
- **Application use cases are the only place transactions are opened.** A use case that touches multiple aggregates wraps them in `pgx.Tx` via a `UnitOfWork` port. Repositories accept the tx/conn through context, not as a global.
- **DTOs at the boundary, never inside the domain.** HTTP request/response structs live in `interfaces/http/.../dto/`. Map domain ↔ DTO at the edge. Never expose `*domain.Deck` over JSON.
- **Domain types are pure Go.** No `db:"..."` tags, no `json:"..."` tags, no Postgres types in domain structs. Pure value objects, entities, aggregates, domain services.
- **Errors**: domain emits typed sentinel errors (`var ErrDeckNotFound = errors.New(...)`); infrastructure wraps DB errors into them; HTTP layer maps them to status codes via a single `errorMapper`.
- **One bounded context = one Postgres schema** (or at minimum a strict table-name prefix). No cross-context joins; if you need data from another context, go through its repository or a read model.

#### Application use case shape

```go
// internal/application/deck/create_deck.go
type CreateDeckInput struct {
    OwnerID string
    Title   string
    Cards   []CardDraft
}
type CreateDeckOutput struct{ DeckID string }

type CreateDeck struct {
    decks  deck.Repository      // domain interface
    ids    ports.IDGenerator
    clock  ports.Clock
    events ports.EventPublisher
    uow    ports.UnitOfWork
}

func (uc *CreateDeck) Handle(ctx context.Context, in CreateDeckInput) (CreateDeckOutput, error) { ... }
```

Use cases are stateless structs; dependencies wired in `cmd/api/main.go`.

### API contract

- REST + JSON (no GraphQL, no gRPC initially). One resource per bounded context.
- OpenAPI 3.1 spec in `docs/backend/openapi.yaml`, kept in sync with handlers. Frontend MSW mocks derive from this spec.
- Auth: `Authorization: Bearer <jwt>`. Refresh token rotation through a dedicated `/auth/refresh` endpoint.
- All write endpoints require idempotency where the operation is non-idempotent (e.g. `Idempotency-Key` header for create-deck).

### Conventions

- `gofmt`, `go vet`, **`golangci-lint`** (config in `.golangci.yaml`) MUST pass before claiming done. Lint includes `errcheck`, `govet`, `staticcheck`, `revive`, `gocritic`, `gosec`, `sqlclosecheck`.
- Test pyramid: lots of pure unit tests on domain (no DB), focused use-case tests with fake repos, integration tests for HTTP+Postgres via testcontainers. Target ≥ 80% coverage on `domain/` and `application/`.
- Migrations are append-only — never edit a merged migration; write a new one.
- All times in UTC; persist as `timestamptz`. All money/scoring as integers (no floats).
- Context propagation: every function below the HTTP layer takes `context.Context` as the first argument, including repository methods.

### Local dev

- `docker-compose.yaml` brings up Postgres on a non-default port.
- `.env.example` lists required vars (`DATABASE_URL`, `JWT_SECRET`, `AI_API_KEY`, ...). Real `.env` is git-ignored.
- `make dev` runs `air` against `cmd/api`. `make migrate-up`, `make migrate-down`, `make sqlc`, `make lint`, `make test`.

---

## Infra

### Workflow: local-first, manual deploy. No git, no CI/CD

- **Develop fully locally** until the app is genuinely ready. Do not ship work-in-progress to the server. No staging environment — when something goes to the server, it is the production deploy.
- **No git on the server.** No remote repository. No CI pipeline. No GitHub Actions / GitLab CI / etc.
- **Deploy = build artifacts locally, push them to the server via SSH/rsync, restart the service.** Source code never lives on the server.
- The server only runs three things: nginx, the Go API binary (as a systemd service), and PostgreSQL.

### Server

- **Host (VDSina):** `146.103.101.232`
- **SSH user:** `root`
- **SSH password:** `FX18H1WZ#Qph7t15L8~x` ⚠️ rotate to SSH-key-only auth as soon as practical (`PasswordAuthentication no` in `sshd_config`); password access is a last-resort fallback.
- **Domain:** `v811467.hosted-by-vdsina.com` — DNS already points to the IP via the hosting provider.
- **OS:** Debian/Ubuntu (assumed). All commands below assume `apt`.

> **Secrets handling:** this `docs/` folder contains real credentials. Per the no-git rule it must **never** be committed anywhere. If a git repo is ever introduced later, scrub `docs/stack.md` and move secrets to a local-only `.env`/`secrets/` directory listed in `.gitignore`. Treat the password as compromised the moment it leaves a trusted machine.

### One-time server bootstrap

Run on the server (SSH in as root):

```bash
apt update && apt upgrade -y
apt install -y nginx postgresql postgresql-contrib certbot python3-certbot-nginx ufw

# Firewall: allow SSH + HTTP + HTTPS only
ufw allow OpenSSH && ufw allow 'Nginx Full' && ufw --force enable

# Postgres: create app DB + user
sudo -u postgres psql <<'SQL'
CREATE USER micocards WITH PASSWORD '<strong-random-password>';
CREATE DATABASE micocards OWNER micocards;
SQL

# App directories
mkdir -p /opt/micocards/{bin,env}
mkdir -p /var/www/micocards          # static frontend (vp build output)
chown -R www-data:www-data /var/www/micocards
```

### SSL: Let's Encrypt via certbot

Free TLS certificate for `v811467.hosted-by-vdsina.com`. Done **after** nginx is configured with the domain in a `server_name` directive.

```bash
certbot --nginx -d v811467.hosted-by-vdsina.com \
        --agree-tos -m <admin-email> --redirect --non-interactive
```

- `--redirect` rewrites the HTTP server block to 301 → HTTPS automatically.
- Certificates live in `/etc/letsencrypt/live/v811467.hosted-by-vdsina.com/`.
- Auto-renewal is installed by the package as a `systemd` timer (`certbot.timer`); verify with `systemctl list-timers | grep certbot` and dry-run with `certbot renew --dry-run`.

### nginx

Single site config at `/etc/nginx/sites-available/micocards` (symlinked into `sites-enabled/`). Serves the SPA build statically and reverse-proxies `/api` to the Go binary on `127.0.0.1:8080`.

```nginx
server {
    listen 80;
    server_name v811467.hosted-by-vdsina.com;
    return 301 https://$host$request_uri;            # certbot may rewrite this
}

server {
    listen 443 ssl http2;
    server_name v811467.hosted-by-vdsina.com;

    ssl_certificate     /etc/letsencrypt/live/v811467.hosted-by-vdsina.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/v811467.hosted-by-vdsina.com/privkey.pem;
    include             /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam         /etc/letsencrypt/ssl-dhparams.pem;

    # Static SPA (vp build output)
    root /var/www/micocards;
    index index.html;

    # PWA: never cache the entry HTML or service worker
    location = /index.html      { add_header Cache-Control "no-store"; try_files $uri =404; }
    location = /sw.js           { add_header Cache-Control "no-store"; try_files $uri =404; }
    location = /manifest.webmanifest { add_header Cache-Control "no-store"; try_files $uri =404; }

    # Hashed assets — long cache
    location /assets/ {
        add_header Cache-Control "public, max-age=31536000, immutable";
        try_files $uri =404;
    }

    # SPA fallback
    location / { try_files $uri $uri/ /index.html; }

    # API
    location /api/ {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
    }

    client_max_body_size 10m;
    gzip on;
    gzip_types text/plain text/css application/json application/javascript application/manifest+json image/svg+xml;
}
```

Reload after edits: `nginx -t && systemctl reload nginx`.

### Backend as a systemd service

`/etc/systemd/system/micocards-api.service`:

```ini
[Unit]
Description=Micocards API
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=www-data
EnvironmentFile=/opt/micocards/env/api.env
ExecStart=/opt/micocards/bin/api
Restart=on-failure
RestartSec=2s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

`/opt/micocards/env/api.env` (chmod 600, owner www-data) holds runtime secrets — **never** baked into the binary or committed:

```
DATABASE_URL=postgres://micocards:<password>@127.0.0.1:5432/micocards?sslmode=disable
JWT_SECRET=<random-64-bytes-base64>
AI_API_KEY=<key>
LISTEN_ADDR=127.0.0.1:8080
```

Lifecycle: `systemctl enable --now micocards-api`, `systemctl restart micocards-api`, `journalctl -u micocards-api -f`.

### Local → server deployment

Always cross-compile from the local machine; the server never sees Go source or `node_modules`.

```bash
# 1. Build artifacts locally
(cd frontend && vp build)                                     # → frontend/build/
(cd backend  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
                go build -trimpath -ldflags="-s -w" \
                -o ./bin/api ./cmd/api)

# 2. Push frontend (atomic-ish via rsync --delete-after)
rsync -avz --delete-after \
      frontend/build/ \
      root@146.103.101.232:/var/www/micocards/

# 3. Push backend binary, then restart
scp backend/bin/api root@146.103.101.232:/opt/micocards/bin/api.new
ssh root@146.103.101.232 '
  mv /opt/micocards/bin/api.new /opt/micocards/bin/api &&
  chmod +x /opt/micocards/bin/api &&
  systemctl restart micocards-api
'

# 4. Migrations (run from local against the remote DB through an SSH tunnel,
#    OR copy the goose binary + migrations and run on-host — see below)
ssh -L 5433:127.0.0.1:5432 root@146.103.101.232 \
    'goose -dir /opt/micocards/migrations postgres "$DATABASE_URL" up'
```

A `Makefile` target `make deploy` should chain these steps. Keep the deploy script idempotent and rerunnable.

### Postgres on the server

- Listens on `127.0.0.1` only (default Debian config) — never expose 5432 publicly.
- `pg_dump` runs nightly to `/var/backups/micocards/` via a systemd timer; copy backups off-host periodically (`rsync` to local on demand).
- Migrations applied through goose during each deploy, before the API binary is restarted (safe-by-default migrations only — additive changes, no destructive DDL on hot tables).

### Operations checklist (per deploy)

1. `make check && make test` locally — refuse to deploy on red.
2. Build frontend + backend artifacts.
3. `rsync` frontend, `scp` backend binary, run migrations.
4. `systemctl restart micocards-api` and tail `journalctl -u micocards-api -f` for 30s.
5. `curl -fsS https://v811467.hosted-by-vdsina.com/api/healthz` — fail the deploy if non-200.
6. Smoke-test the SPA (open the deployed URL in a real browser).
