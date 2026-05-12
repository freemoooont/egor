# Use Cases

One section per application use case. Every section lists Input DTO, Output DTO,
preconditions, invariants enforced (referencing aggregate invariants by number),
side effects, events emitted, error sentinels (mapped to HTTP via the central
`errorMapper` — see ADR 0006), the idempotency story, and the HTTP endpoint that
exposes it.

DTOs live in `interfaces/http/.../dto/`. Application input/output structs live in
`application/<context>/`. They are distinct on purpose; the boundary mapper bridges
them.

Every "events emitted" entry below names a `DomainEvent` defined in
`docs/backend/domain-events.md`; each one is persisted as an `OutboxRecord` in the
emitting context's `outbox` table inside the same `pgx.Tx` as the use case (see ADR
0002). Non-idempotent writes accept an `IdempotencyKey` (HTTP `Idempotency-Key`
header) per ADR 0005 — the value is recorded in `iam.idempotency_keys` and replays
return the cached response.

---

## iam

### `RegisterUser`

- **HTTP:** `POST /api/auth/register`
- **Input:** `{ Email: string, Password: string, DisplayName: string }`
- **Output:** `{ UserID: string, AccessToken: string, RefreshToken: string, AccessTokenExpiresAt: time.Time, RefreshTokenExpiresAt: time.Time, User: { ID, Email, DisplayName, AvatarRef } }`
- **Preconditions:** none (anonymous endpoint).
- **Invariants enforced:** `User_EmailAddress*` (1–4), `User_PasswordHashMustComeFromBcryptHasher` (5), `User_DisplayNameLengthBetween1And64` (6), `User_AvatarRefDefaultsToNoneOnRegistration` (11).
- **Side effects:** insert `iam.users` row, insert initial `RefreshTokenFamily` + `RefreshToken` rows, write outbox.
- **Events emitted:** `UserRegistered`, `RefreshTokenIssued`.
- **Error sentinels:** `ErrInvalidEmail` (422), `ErrInvalidDisplayName` (422), `ErrPasswordTooWeak` (422), `ErrEmailTaken` (409).
- **Idempotency:** accepts `Idempotency-Key`; replay returns the cached response (same access+refresh) — see ADR 0005.

### `LoginUser`

- **HTTP:** `POST /api/auth/login`
- **Input:** `{ Email: string, Password: string }`
- **Output:** `{ AccessToken, RefreshToken, AccessTokenExpiresAt, RefreshTokenExpiresAt, User }`
- **Preconditions:** none (anonymous endpoint).
- **Invariants enforced:** `User_EmailAddressIsLowerCasedOnConstruction` (3) on lookup; bcrypt verification.
- **Side effects:** insert new `RefreshTokenFamily` + `RefreshToken`, write outbox.
- **Events emitted:** `UserLoggedIn`, `RefreshTokenIssued`.
- **Error sentinels:** `ErrInvalidCredentials` (401), `ErrInvalidEmail` (422).
- **Idempotency:** not idempotent (every successful login mints a fresh family); the endpoint does **not** read `Idempotency-Key`.

### `RefreshAccessToken`

- **HTTP:** `POST /api/auth/refresh`
- **Input:** `{ RefreshToken: string }`
- **Output:** `{ AccessToken, RefreshToken, AccessTokenExpiresAt, RefreshTokenExpiresAt }`
- **Preconditions:** valid `RefreshToken` belonging to a non-revoked family.
- **Invariants enforced:** `RefreshTokenFamily_OnlyTheLatestTokenIsValid` (9), `RefreshTokenFamily_ExpiredTokenCannotMint` (10), `RefreshTokenFamily_TokensAreOrderedByIssuedAt` (8).
- **Side effects:** revoke the consumed token (or the entire family on reuse detection), insert the rotated token, write outbox.
- **Events emitted:** `RefreshTokenRevoked`, `RefreshTokenIssued`.
- **Error sentinels:** `ErrRefreshTokenInvalid` (401), `ErrRefreshTokenExpired` (401), `ErrRefreshTokenReused` (401, with family revoke side effect).
- **Idempotency:** not exposed via header; the rotation itself is single-use by construction.

### `LogoutUser`

- **HTTP:** `POST /api/auth/logout`
- **Input:** `{ RefreshToken: string }` (request body) — caller may also be on a valid access token.
- **Output:** `{ ok: true }`
- **Preconditions:** authenticated (`Authorization: Bearer ...`).
- **Invariants enforced:** none beyond ownership of the supplied token.
- **Side effects:** revoke the supplied refresh token's family, write outbox.
- **Events emitted:** `RefreshTokenRevoked`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrRefreshTokenInvalid` (404).
- **Idempotency:** naturally idempotent; repeated calls on an already-revoked family are no-ops returning `{ ok: true }`.

### `GetCurrentUser`

- **HTTP:** `GET /api/me`
- **Input:** none (user id from JWT claims).
- **Output:** `{ ID, Email, DisplayName, AvatarRef, RegisteredAt }`
- **Preconditions:** authenticated.
- **Invariants enforced:** none (read).
- **Side effects:** none.
- **Events emitted:** none.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrUserNotFound` (404 — defensive; should not happen for a valid JWT).
- **Idempotency:** GET — naturally idempotent.

### `UpdateProfile`

- **HTTP:** `PATCH /api/me`
- **Input:** `{ DisplayName?: string, Email?: string }`
- **Output:** `{ ID, Email, DisplayName, AvatarRef }`
- **Preconditions:** authenticated.
- **Invariants enforced:** `User_EmailAddress*` (1–4) on email change, `User_DisplayNameLengthBetween1And64` (6).
- **Side effects:** update `iam.users` row.
- **Events emitted:** none in v1 (event hooks reserved).
- **Error sentinels:** `ErrUnauthorized` (401), `ErrInvalidEmail` (422), `ErrInvalidDisplayName` (422), `ErrEmailTaken` (409).
- **Idempotency:** PATCH with the same payload is naturally idempotent; the endpoint does not read `Idempotency-Key`.

### `ChangePassword`

- **HTTP:** `POST /api/me/password`
- **Input:** `{ CurrentPassword: string, NewPassword: string }`
- **Output:** `{ ok: true }`
- **Preconditions:** authenticated.
- **Invariants enforced:** `User_PasswordHashMustComeFromBcryptHasher` (5), `User_ChangePasswordRotatesHashAndRevokesAllRefreshFamilies` (7).
- **Side effects:** rotate `iam.users.password_hash`, revoke every active `RefreshTokenFamily` for the user, write outbox.
- **Events emitted:** `RefreshTokenRevoked` (one per active family).
- **Error sentinels:** `ErrUnauthorized` (401), `ErrInvalidCredentials` (401), `ErrPasswordTooWeak` (422).
- **Idempotency:** repeated calls succeed but rotate the hash again; clients must not retry blindly. No `Idempotency-Key` support — manual confirmation in UI.

### `UploadAvatar`

- **HTTP:** `POST /api/me/avatar`
- **Status in v1:** **stub**. The handler returns HTTP 501 with body
  `{"error":"not_implemented","message":"avatar upload deferred"}` per the spec
  (Account Settings displays initials on brand orange in the meantime).
- **Input (when implemented):** multipart `file` (PNG/JPEG ≤2MB).
- **Output (when implemented):** `{ AvatarRef: string }`
- **Preconditions:** authenticated.
- **Invariants enforced (when implemented):** content-type whitelist, size cap.
- **Side effects (when implemented):** upload to object store, set `AvatarRef`.
- **Events emitted:** none planned.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrNotImplemented` (501).
- **Idempotency:** `Idempotency-Key` accepted when implemented (see ADR 0005).

---

## decks

### `CreateDeck`

- **HTTP:** `POST /api/decks`
- **Input:** `{ Title: string, Cards?: [{ Term: string, Definition: string }] }`
- **Output:** `{ DeckID: string, Title: string, Cards: [{ ID, Term, Definition, Ordinal }], CreatedAt: time.Time }`
- **Preconditions:** authenticated; owner derived from JWT.
- **Invariants enforced:** `Deck_TitleLengthBetween1And120` (1), `Deck_HasAtMost500Cards` (3), `Card_TermLengthBetween1And512` (6), `Card_DefinitionLengthBetween1And2048` (7), `Deck_CardOrdinalsAreDense1ToN` (4), `Deck_CardOrdinalsAreUnique` (5).
- **Side effects:** insert `decks.decks` row + N `decks.cards` rows + outbox in one tx.
- **Events emitted:** `DeckCreated`, plus one `CardAdded` per card.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrInvalidDeckTitle` (422), `ErrDeckTitleTooLong` (422), `ErrInvalidTerm` (422), `ErrInvalidDefinition` (422), `ErrDeckCardLimitExceeded` (422).
- **Idempotency:** `Idempotency-Key` required for the cards-included form. Replay returns the cached response.

### `RenameDeck`

- **HTTP:** `PATCH /api/decks/{deckID}`
- **Input:** `{ Title: string }`
- **Output:** `{ DeckID, Title, RenamedAt }`
- **Preconditions:** authenticated; caller owns the deck.
- **Invariants enforced:** `Deck_TitleLengthBetween1And120` (1), `Deck_OnlyOwnerCanMutate` (13), `Deck_DeleteIsTerminal` (12), `Deck_RenameDoesNotMutateCards` (11).
- **Side effects:** update `decks.decks.title` + outbox.
- **Events emitted:** `DeckRenamed`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404), `ErrInvalidDeckTitle` (422), `ErrDeckTitleTooLong` (422), `ErrDeckDeleted` (409).
- **Idempotency:** PATCH with the same title is naturally idempotent.

### `DeleteDeck`

- **HTTP:** `DELETE /api/decks/{deckID}`
- **Input:** none.
- **Output:** `{ ok: true }`
- **Preconditions:** authenticated; caller owns the deck.
- **Invariants enforced:** `Deck_OnlyOwnerCanMutate` (13), `Deck_DeleteIsTerminal` (12).
- **Side effects:** soft-delete `decks.decks` row, hard-delete `decks.cards`, write outbox. Subscribers in `practice` clean up.
- **Events emitted:** `DeckDeleted`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404).
- **Idempotency:** repeated DELETE is a no-op returning `{ ok: true }`.

### `AddCard`

- **HTTP:** `POST /api/decks/{deckID}/cards`
- **Input:** `{ Term: string, Definition: string }`
- **Output:** `{ CardID, Term, Definition, Ordinal }`
- **Preconditions:** authenticated; caller owns the deck; deck not deleted.
- **Invariants enforced:** `Deck_HasAtMost500Cards` (3), `Card_TermLengthBetween1And512` (6), `Card_DefinitionLengthBetween1And2048` (7), `Deck_AddCardAppendsAtNextOrdinal` (8), `Deck_OnlyOwnerCanMutate` (13).
- **Side effects:** insert `decks.cards` + outbox.
- **Events emitted:** `CardAdded`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404), `ErrDeckDeleted` (409), `ErrDeckCardLimitExceeded` (422), `ErrInvalidTerm` (422), `ErrInvalidDefinition` (422).
- **Idempotency:** `Idempotency-Key` required.

### `EditCard`

- **HTTP:** `PATCH /api/decks/{deckID}/cards/{cardID}`
- **Input:** `{ Term?: string, Definition?: string }`
- **Output:** `{ CardID, Term, Definition, Ordinal }`
- **Preconditions:** authenticated; caller owns the deck; card belongs to the deck.
- **Invariants enforced:** `Card_TermLengthBetween1And512` (6), `Card_DefinitionLengthBetween1And2048` (7), `Deck_OnlyOwnerCanMutate` (13).
- **Side effects:** update `decks.cards` + outbox.
- **Events emitted:** `CardEdited`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404), `ErrCardNotFound` (404), `ErrInvalidTerm` (422), `ErrInvalidDefinition` (422).
- **Idempotency:** PATCH is naturally idempotent.

### `RemoveCard`

- **HTTP:** `DELETE /api/decks/{deckID}/cards/{cardID}`
- **Input:** none.
- **Output:** `{ ok: true }`
- **Preconditions:** authenticated; caller owns the deck.
- **Invariants enforced:** `Deck_RemoveCardRecompactsOrdinals` (9), `Deck_OnlyOwnerCanMutate` (13).
- **Side effects:** delete `decks.cards` row, recompact remaining ordinals, write outbox.
- **Events emitted:** `CardRemoved`, `CardsReordered` (when recompaction reorders the rest).
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404), `ErrCardNotFound` (404).
- **Idempotency:** repeated DELETE is a no-op.

### `ReorderCards`

- **HTTP:** `PUT /api/decks/{deckID}/cards/order`
- **Input:** `{ OrderedIDs: [string] }` — full permutation of the deck's card ids.
- **Output:** `{ Cards: [{ ID, Term, Definition, Ordinal }] }`
- **Preconditions:** authenticated; caller owns the deck.
- **Invariants enforced:** `Deck_ReorderCardsRequiresExactPermutationOfExistingIDs` (10), `Deck_CardOrdinalsAreDense1ToN` (4), `Deck_CardOrdinalsAreUnique` (5), `Deck_OnlyOwnerCanMutate` (13).
- **Side effects:** rewrite ordinals on every `decks.cards` row, write outbox.
- **Events emitted:** `CardsReordered`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404), `ErrInvalidCardReorder` (422).
- **Idempotency:** PUT with the same ordering is naturally idempotent.

### `GetDeck`

- **HTTP:** `GET /api/decks/{deckID}`
- **Input:** none.
- **Output:** `{ DeckID, Title, Cards: [{ ID, Term, Definition, Ordinal }], CreatedAt, OwnerID }`
- **Preconditions:** authenticated; caller owns the deck.
- **Invariants enforced:** none (read), but ownership check applies (`Deck_OnlyOwnerCanMutate` is the write-side; reads enforce the same gate).
- **Side effects:** none.
- **Events emitted:** none.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404).
- **Idempotency:** GET — naturally idempotent.

### `ListUserDecks`

- **HTTP:** `GET /api/decks`
- **Input:** query `{ Limit?: int, Cursor?: string }`.
- **Output:** `{ Decks: [{ DeckID, Title, CardCount, CreatedAt }], NextCursor?: string }`
- **Preconditions:** authenticated.
- **Invariants enforced:** none (read).
- **Side effects:** none.
- **Events emitted:** none.
- **Error sentinels:** `ErrUnauthorized` (401).
- **Idempotency:** GET — naturally idempotent.

### `GenerateDeckWithAI`

- **HTTP:** `POST /api/decks/generate`
- **Status in v1:** **stub**. Returns HTTP 501 with body
  `{"error":"not_implemented","message":"AI generation deferred"}` when `AI_API_KEY`
  is unset (the default per spec assumption). Skeleton, DTOs, OpenAPI entry, and the
  events are wired so v2 can fill it in.
- **Input:** `{ Prompt: string }`
- **Output (when implemented):** `{ Draft: AIDeckDraft }` where `AIDeckDraft = { Title, Cards: [{ Term, Definition }] }`.
- **Preconditions:** authenticated.
- **Invariants enforced:** input length cap; output validates as a `Deck` candidate.
- **Side effects:** request to upstream AI API; no DB writes.
- **Events emitted:** `DeckGenerationRequested`, `DeckGenerationCompleted` (with `status="not_implemented"` while stubbed).
- **Error sentinels:** `ErrUnauthorized` (401), `ErrNotImplemented` (501), `ErrAIUpstream` (502, when implemented).
- **Idempotency:** `Idempotency-Key` required (drives `RequestID`). Replay returns the cached response.

---

## practice

### `StartPracticeSession`

- **HTTP:** `POST /api/practice/sessions`
- **Input:** `{ DeckID: string, Mode: "tracked" | "untracked" }`
- **Output:** `{ SessionID, DeckID, Mode, CardIDs: [string], StartedAt }`
- **Preconditions:** authenticated; caller owns the deck; deck has ≥1 card.
- **Invariants enforced:** `Session_StatusStartsAsInProgress` (1), `Session_OwnedBySingleUser` (8), `Session_DeckIDIsImmutable` (9).
- **Side effects:** snapshot card ids from `decks.Decks` repo, insert `practice.sessions` + snapshot + outbox.
- **Events emitted:** `PracticeSessionStarted`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404), `ErrDeckEmpty` (422), `ErrInvalidPracticeMode` (422).
- **Idempotency:** `Idempotency-Key` required. Replay returns the existing session.

### `RateCard`

- **HTTP:** `POST /api/practice/sessions/{sessionID}/ratings`
- **Input:** `{ CardID: string, Rating: 0 | 1 | 2 }`
- **Output:** `{ SessionID, CardID, Rating, RatedAt }`
- **Preconditions:** authenticated; caller owns the session; session is `InProgress`; mode is `Tracked`.
- **Invariants enforced:** `Session_RateRequiresInProgressStatus` (2), `Session_RateRequiresTrackedMode` (3), `Session_RateRejectsCardsNotInSnapshot` (4), `Session_RateIsIdempotentPerCardID` (5).
- **Side effects:** upsert `practice.rated_cards` row, write outbox.
- **Events emitted:** `CardRated`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrSessionNotFound` (404), `ErrSessionClosed` (409), `ErrSessionUntracked` (409), `ErrCardNotInSession` (422), `ErrInvalidRating` (422).
- **Idempotency:** natural per-card idempotency via the upsert; no header required.

### `FinishPracticeSession`

- **HTTP:** `POST /api/practice/sessions/{sessionID}/finish`
- **Input:** none.
- **Output:** `{ SessionID, Mode, CountDontKnow, CountStillLearning, CountKnowKnow, CompletedAt }`
- **Preconditions:** authenticated; caller owns the session.
- **Invariants enforced:** `Session_FinishIsTerminalAndIdempotent` (6), `Session_TrackedFinishEmitsCardRatedSummary` (10).
- **Side effects:** flip `status` to `Completed`, stamp `completed_at`, write outbox; in-process handler updates `UserDeckProgress`.
- **Events emitted:** `PracticeSessionCompleted`.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrSessionNotFound` (404).
- **Idempotency:** terminal+idempotent by aggregate invariant 6.

### `GetPracticeResults`

- **HTTP:** `GET /api/practice/sessions/{sessionID}/results`
- **Input:** none.
- **Output:** `{ SessionID, DeckID, Mode, CountDontKnow, CountStillLearning, CountKnowKnow, RatedCards: [{ CardID, Rating }], CompletedAt }`
- **Preconditions:** authenticated; caller owns the session; session is `Completed`.
- **Invariants enforced:** none (read).
- **Side effects:** none.
- **Events emitted:** none.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrSessionNotFound` (404), `ErrSessionNotCompleted` (409).
- **Idempotency:** GET — naturally idempotent.

### `GetUserDeckProgress`

- **HTTP:** `GET /api/decks/{deckID}/progress`
- **Input:** none (user from JWT).
- **Output:** `{ DeckID, CardProgress: [{ CardID, Rating, LastRatedAt }] }`
- **Preconditions:** authenticated; caller owns the deck.
- **Invariants enforced:** none (read; backed by `UserDeckProgress` projection).
- **Side effects:** none.
- **Events emitted:** none.
- **Error sentinels:** `ErrUnauthorized` (401), `ErrForbidden` (403), `ErrDeckNotFound` (404).
- **Idempotency:** GET — naturally idempotent.
