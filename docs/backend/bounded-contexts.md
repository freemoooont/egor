# Bounded Contexts

Three contexts, one Postgres schema each. No cross-schema joins. Cross-context reads
go through the owning context's repository or a public read model. Cross-context
writes are HTTP-only between deployable units (today there is one deployable, but the
boundaries are drawn so that splitting later requires no domain rewrite).

```
internal/
├── domain/iam        application/iam        infrastructure/postgres/iamrepo        interfaces/http/iam
├── domain/decks      application/decks      infrastructure/postgres/decksrepo      interfaces/http/decks
└── domain/practice   application/practice   infrastructure/postgres/practicerepo   interfaces/http/practice
```

Postgres schemas: `iam`, `decks`, `practice`. Migrations live in
`backend/migrations/<context>/*.up.sql` / `*.down.sql`. sqlc input lives in
`backend/queries/<context>/*.sql`.

---

## Context: `iam`

**Purpose.** Authenticate users and gate access. Owns identity, credentials, profile,
refresh-token rotation, and the cross-cutting idempotency-key store.

**Aggregates owned.**
- `iam.User` — root; child VOs `EmailAddress`, `PasswordHash`, `DisplayName`,
  `AvatarRef`; child entity `RefreshTokenFamily`.

**Repositories owned (interfaces in `internal/domain/iam/`).**
- `Users` — `Save`, `ByID`, `ByEmail`, `EmailExists`.
- `RefreshTokens` — `Save`, `ByOpaqueValue`, `RevokeFamily`, `RevokeOne`.
- `IdempotencyKeys` — `Get`, `Put` (used by `application/idempotency` middleware).

**Postgres schema name.** `iam`. Tables: `iam.users`, `iam.refresh_tokens`,
`iam.idempotency_keys`, `iam.outbox`.

**Public domain events emitted.**
- `UserRegistered`
- `UserLoggedIn`
- `RefreshTokenIssued`
- `RefreshTokenRevoked`

**Public domain events consumed.** None (root of the dependency graph).

**Integration boundary.** Other contexts identify a user only by `user_id` (UUID). They
must never read `iam.users` directly. To resolve a `user_id` → display data they call
the `iam` HTTP API. There is no cross-schema join.

---

## Context: `decks`

**Purpose.** Authoring and lifecycle of decks and their cards. Owns the catalogue of
decks each user has built. The AI-generation stub also lives here.

**Aggregates owned.**
- `decks.Deck` — root; child VOs `DeckTitle`, `OwnerID`; child entity `Card` (which
  itself has VOs `Term`, `Definition`, `CardOrdinal`).

**Repositories owned (interfaces in `internal/domain/decks/`).**
- `Decks` — `Save`, `ByID`, `ByOwner`, `Delete`.

**Postgres schema name.** `decks`. Tables: `decks.decks`, `decks.cards`,
`decks.outbox`.

**Public domain events emitted.**
- `DeckCreated`
- `DeckRenamed`
- `DeckDeleted`
- `CardAdded`
- `CardEdited`
- `CardRemoved`
- `CardsReordered`
- `DeckGenerationRequested`
- `DeckGenerationCompleted`

**Public domain events consumed.**
- `UserRegistered` (no-op today; reserved hook for future welcome-deck seeding).

**Integration boundary.** Authorises every write against the `OwnerID` carried in the
JWT claims (resolved by the HTTP middleware in `iam`). Validates that the owner exists
by trusting the JWT — no synchronous call into `iam`. The `practice` context references
`Deck` only by `deck_id`; if it needs the deck contents it calls the `decks` HTTP API
or its repository through composition root, never via cross-schema join.

---

## Context: `practice`

**Purpose.** Run a practice pass over a deck and aggregate per-user-per-deck progress.

**Aggregates owned.**
- `practice.Session` — root; child entity `RatedCard`; VOs `PracticeSessionMode`,
  `PracticeSessionStatus`, `CardRating`.
- `practice.UserDeckProgress` — read-model aggregate; VO `CardProgress`.

**Repositories owned (interfaces in `internal/domain/practice/`).**
- `Sessions` — `Save`, `ByID`, `LatestCompletedFor`.
- `UserDeckProgresses` — `Save`, `ByUserAndDeck`.

**Postgres schema name.** `practice`. Tables: `practice.sessions`,
`practice.rated_cards`, `practice.user_deck_progress`, `practice.outbox`.

**Public domain events emitted.**
- `PracticeSessionStarted`
- `CardRated`
- `PracticeSessionCompleted`

**Public domain events consumed.**
- `DeckDeleted` — cascades into soft-archive of any open `Session` for that deck and
  removal of `UserDeckProgress` rows for that deck.
- `CardRemoved` — drops the corresponding `RatedCard` rows from open sessions and
  `CardProgress` entries from the read model.

**Integration boundary.** When starting a session, takes `deck_id` and a snapshot of
the deck's card ids from the `decks` repository (read-only). All writes happen in the
`practice` schema. Never joins to `decks.cards` or `iam.users`.

---

## Dependency direction (Mermaid)

```mermaid
graph LR
    iam[iam<br/>users, auth, idempotency]
    decks[decks<br/>decks, cards, AI stub]
    practice[practice<br/>sessions, progress]

    decks -->|reads user_id<br/>from JWT issued by iam| iam
    practice -->|reads deck_id + card_ids<br/>via decks.Decks repo at session start| decks
    practice -->|reads user_id<br/>from JWT issued by iam| iam

    decks -.->|consumes UserRegistered<br/>(reserved, no-op today)| iam
    practice -.->|consumes DeckDeleted, CardRemoved<br/>via in-process bus + outbox| decks
```

Edges are one-way only. `iam` depends on no other context. The dotted edges are domain
event subscriptions (in-process synchronous today; outbox-ready — see ADR 0002).
