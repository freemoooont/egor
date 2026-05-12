# Ubiquitous Language

This glossary is the single source of truth for domain vocabulary across product, design,
and code. Names in Go code MUST match these canonical English terms verbatim — no
synonyms, no translations. Russian aliases in parentheses are for designer/PM
communication only and never appear in identifiers.

Every term below is load-bearing: it is referenced from at least one aggregate
(`aggregates.md`), use case (`use-cases.md`), or domain event (`domain-events.md`).

---

## Bounded context: `iam`

Identity and access management. Owns users, credentials, and the refresh-token family.

| Term | Russian alias | Definition | Lives in |
|---|---|---|---|
| **User** | пользователь | Aggregate root in `iam`. Authenticated principal that owns decks and practice sessions. Identified by a UUID. | aggregate `iam.User`; events `UserRegistered`, `UserLoggedIn`; use cases `RegisterUser`, `LoginUser`, `GetCurrentUser`, `UpdateProfile`, `ChangePassword`, `UploadAvatar` |
| **EmailAddress** | электронная почта | Value object inside `User`. Lower-cased, RFC-5321-validated string; unique across all users. | VO inside `iam.User`; use cases `RegisterUser`, `LoginUser`, `UpdateProfile` |
| **PasswordHash** | хеш пароля | Value object inside `User`. Opaque bcrypt digest of the plaintext password. Plaintext is never persisted, logged, or returned. | VO inside `iam.User`; use cases `RegisterUser`, `LoginUser`, `ChangePassword` |
| **DisplayName** | имя | Value object inside `User`. Human-friendly profile name shown on the Account screen. 1..64 graphemes after trim. | VO inside `iam.User`; use case `UpdateProfile` |
| **AvatarRef** | аватар | Value object inside `User`. Either `none` (rendered client-side as initials on brand orange) or a server-side reference returned by the avatar upload stub. | VO inside `iam.User`; use case `UploadAvatar` |
| **AccessToken** | access-токен | Stateless 15-minute JWT (HS256) signed with `JWT_SECRET`. Carries `sub=user_id`, `exp`, `iat`. Verified by HTTP middleware. | use cases `LoginUser`, `RefreshAccessToken`; ADR 0003 |
| **RefreshToken** | refresh-токен | 7-day rotating opaque token persisted in `iam.refresh_tokens`. Issued alongside an `AccessToken`. Single-use: rotated on every refresh. | aggregate `iam.User` (via `RefreshTokenFamily`); events `RefreshTokenIssued`, `RefreshTokenRevoked`; use cases `LoginUser`, `RefreshAccessToken`, `LogoutUser` |
| **RefreshTokenFamily** | семейство refresh-токенов | Linked chain of `RefreshToken`s rooted at a single login. Reuse of a revoked token revokes the entire family (compromise containment). | child of `iam.User`; events `RefreshTokenIssued`, `RefreshTokenRevoked`; ADR 0003 |
| **IdempotencyKey** | ключ идемпотентности | Client-supplied opaque string in the `Idempotency-Key` header. Scoped per-endpoint. Replays return the cached response. | persisted in `iam.idempotency_keys`; use cases `RegisterUser`, `CreateDeck`, `StartPracticeSession`; ADR 0005 |

## Bounded context: `decks`

Deck and card authoring. Owns the catalogue of decks each user has built or imported.

| Term | Russian alias | Definition | Lives in |
|---|---|---|---|
| **Deck** | колода | Aggregate root in `decks`. Ordered collection of `Card`s owned by exactly one `User`. | aggregate `decks.Deck`; events `DeckCreated`, `DeckRenamed`, `DeckDeleted`, `DeckGenerationRequested`, `DeckGenerationCompleted`; use cases `CreateDeck`, `RenameDeck`, `DeleteDeck`, `GetDeck`, `ListUserDecks`, `GenerateDeckWithAI` |
| **DeckTitle** | название колоды | Value object inside `Deck`. 1..120 graphemes after trim. Required at creation. | VO inside `decks.Deck`; use cases `CreateDeck`, `RenameDeck` |
| **OwnerID** | владелец | Reference (UUID) to the `iam.User` that owns this `Deck`. Immutable. | VO inside `decks.Deck`; use cases `CreateDeck`, `ListUserDecks` |
| **Card** | карточка | Entity inside the `Deck` aggregate. Pair of `Term` + `Definition` with an `CardOrdinal`. Mutated only through its parent `Deck`. | entity inside `decks.Deck`; events `CardAdded`, `CardEdited`, `CardRemoved`, `CardsReordered`; use cases `AddCard`, `EditCard`, `RemoveCard`, `ReorderCards` |
| **Term** | термин | Value object inside `Card`. The "front" of the flashcard. 1..512 graphemes after trim. | VO inside `Card`; use cases `AddCard`, `EditCard` |
| **Definition** | определение | Value object inside `Card`. The "back" of the flashcard. 1..2048 graphemes after trim. | VO inside `Card`; use cases `AddCard`, `EditCard` |
| **CardOrdinal** | порядковый номер карточки | Value object inside `Card`. 1-based dense integer position within the parent `Deck`. Unique within the deck. | VO inside `Card`; use case `ReorderCards`; event `CardsReordered` |
| **AIDeckDraft** | черновик ИИ-колоды | Value object returned by the AI generation stub: a not-yet-persisted deck title plus a list of `(Term, Definition)` pairs. The user reviews and saves it as a real `Deck`. | VO; events `DeckGenerationRequested`, `DeckGenerationCompleted`; use case `GenerateDeckWithAI` |

## Bounded context: `practice`

Practice sessions and per-deck progress. Reads users and decks via integration; never
joins to their tables directly.

| Term | Russian alias | Definition | Lives in |
|---|---|---|---|
| **PracticeSession** | сессия практики | Aggregate root in `practice`. One pass over a deck by a user. Holds a `PracticeSessionMode`, a list of `RatedCard`s, and a lifecycle (`InProgress` → `Completed` or `Abandoned`). | aggregate `practice.Session`; events `PracticeSessionStarted`, `CardRated`, `PracticeSessionCompleted`; use cases `StartPracticeSession`, `RateCard`, `FinishPracticeSession`, `GetPracticeResults` |
| **PracticeSessionMode** | режим практики | Value object inside `PracticeSession`. Either `Tracked` (rating buttons enabled, results contribute to progress) or `Untracked` (browse-only). Set once at start. | VO inside `practice.Session`; use case `StartPracticeSession` |
| **PracticeSessionStatus** | статус сессии | Value object inside `PracticeSession`. One of `InProgress`, `Completed`, `Abandoned`. Closes once. | VO inside `practice.Session`; use case `FinishPracticeSession` |
| **CardRating** | оценка карточки | Value object enum: `DontKnow=0`, `StillLearning=1`, `KnowKnow=2`. The three buckets the user picks per card in `Tracked` mode. Stored as `smallint`. | VO inside `RatedCard`; event `CardRated`; use case `RateCard` |
| **RatedCard** | оценённая карточка | Entity inside `practice.Session`. Tuple of `(card_id, CardRating, rated_at)`. At most one per `card_id` per session — re-rating updates the existing entry. | entity inside `practice.Session`; event `CardRated` |
| **CardProgress** | прогресс карточки | Value object inside `UserDeckProgress`. Latest `CardRating` for a `(user_id, deck_id, card_id)` triple, last updated by a completed `Tracked` session. | VO inside `practice.UserDeckProgress`; use case `GetUserDeckProgress` |
| **UserDeckProgress** | прогресс по колоде | Read-model aggregate in `practice`. One row per `(user_id, deck_id)` summarising the latest `CardProgress` distribution. Fed by `PracticeSessionCompleted` events. | aggregate `practice.UserDeckProgress`; use case `GetUserDeckProgress` |
| **PracticeResults** | результаты практики | Value object derived from a `Completed` `PracticeSession`. Counts of each `CardRating` bucket plus the rated-card list, used to render the results pie chart. | use case `GetPracticeResults` |

## Cross-context

| Term | Russian alias | Definition | Lives in |
|---|---|---|---|
| **DomainEvent** | доменное событие | Immutable past-tense record that the domain emits when an aggregate's state changes. Carries `event_id`, `event_name`, `occurred_at`, `aggregate_id`, payload. | every event in `domain-events.md`; ADR 0002 |
| **OutboxRecord** | запись outbox | Persisted row in the `outbox` table inside each context's schema. Mirrors a `DomainEvent` plus delivery status. Lets us move from synchronous in-process dispatch to async transport without rewriting the domain. | infrastructure persistence of every event; ADR 0002 |
