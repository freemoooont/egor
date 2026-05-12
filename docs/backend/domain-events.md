# Domain Events

Every event is dispatched in-process synchronously today (see ADR 0002), but every
event is **also** persisted to the per-context `outbox` table in the same `pgx.Tx`
that produced it. Migrating to async transport later requires changing the publisher,
not the domain.

For each event:
- **Name** — past-tense, PascalCase. Matches the Go type name verbatim.
- **Emitted by** — use case (publisher).
- **Payload** — Go struct sketch. UUIDs are `string` in pseudocode; in real code they
  are typed `uuid.UUID`. Times are UTC `time.Time`.
- **Consumers** — in-process subscribers.
- **Delivery** — `sync` (in-process synchronous within the same tx) or `outbox-ready`
  (also persisted to outbox; can be flipped to async without code change).
- **Idempotency key** — what makes a redelivered copy a no-op.

All events share an envelope:

```go
type EventEnvelope struct {
    EventID     string    // UUID v4 — also the idempotency key for the outbox
    EventName   string    // e.g. "decks.DeckCreated"
    OccurredAt  time.Time // UTC
    AggregateID string    // root id (or composite for read models)
    Payload     any       // one of the structs below
}
```

Delivery for every event below is `sync + outbox-ready`.

---

## iam

### `UserRegistered`

- **Emitted by:** `RegisterUser`.
- **Payload:**
  ```go
  type UserRegistered struct {
      UserID       string
      Email        string // already lower-cased
      DisplayName  string
      RegisteredAt time.Time
  }
  ```
- **Consumers:** none in v1 (reserved hook in `decks` for welcome-deck seeding).
- **Idempotency key:** `EventID` (one-shot per registration).

### `UserLoggedIn`

- **Emitted by:** `LoginUser`.
- **Payload:**
  ```go
  type UserLoggedIn struct {
      UserID    string
      LoggedInAt time.Time
      UserAgent string // truncated to 256
      IP        string // RFC1918-aware
  }
  ```
- **Consumers:** none today (audit-log subscriber is an open hook).
- **Idempotency key:** `EventID`.

### `RefreshTokenIssued`

- **Emitted by:** `LoginUser`, `RefreshAccessToken`.
- **Payload:**
  ```go
  type RefreshTokenIssued struct {
      UserID    string
      FamilyID  string
      TokenID   string
      IssuedAt  time.Time
      ExpiresAt time.Time // IssuedAt + 7d
  }
  ```
- **Consumers:** none today.
- **Idempotency key:** `TokenID` (unique per token).

### `RefreshTokenRevoked`

- **Emitted by:** `RefreshAccessToken` (rotation), `LogoutUser`, `ChangePassword`,
  `RegisterUser`-after-reuse-detection.
- **Payload:**
  ```go
  type RefreshTokenRevoked struct {
      UserID    string
      FamilyID  string
      TokenID   string  // empty if RevokeFamily
      Reason    string  // "rotation" | "logout" | "password-change" | "reuse-detected"
      RevokedAt time.Time
  }
  ```
- **Consumers:** none today.
- **Idempotency key:** `(FamilyID, TokenID, Reason)`.

---

## decks

### `DeckCreated`

- **Emitted by:** `CreateDeck`.
- **Payload:**
  ```go
  type DeckCreated struct {
      DeckID    string
      OwnerID   string
      Title     string
      CardCount int
      CreatedAt time.Time
  }
  ```
- **Consumers:** none today (read-model warm-start hook reserved).
- **Idempotency key:** `DeckID`.

### `DeckRenamed`

- **Emitted by:** `RenameDeck`.
- **Payload:**
  ```go
  type DeckRenamed struct {
      DeckID    string
      OwnerID   string
      OldTitle  string
      NewTitle  string
      RenamedAt time.Time
  }
  ```
- **Consumers:** none today.
- **Idempotency key:** `EventID`.

### `DeckDeleted`

- **Emitted by:** `DeleteDeck`.
- **Payload:**
  ```go
  type DeckDeleted struct {
      DeckID    string
      OwnerID   string
      DeletedAt time.Time
  }
  ```
- **Consumers:** `practice.UserDeckProgress` projection (drops all rows for this
  deck); `practice.Sessions` cleanup handler (abandons any open session against
  this deck).
- **Idempotency key:** `DeckID`.

### `CardAdded`

- **Emitted by:** `AddCard`, also one event per card emitted by `CreateDeck` if cards
  were supplied in the create payload.
- **Payload:**
  ```go
  type CardAdded struct {
      DeckID     string
      CardID     string
      Term       string
      Definition string
      Ordinal    int
      AddedAt    time.Time
  }
  ```
- **Consumers:** none today.
- **Idempotency key:** `CardID`.

### `CardEdited`

- **Emitted by:** `EditCard`.
- **Payload:**
  ```go
  type CardEdited struct {
      DeckID         string
      CardID         string
      OldTerm        string
      NewTerm        string
      OldDefinition  string
      NewDefinition  string
      EditedAt       time.Time
  }
  ```
- **Consumers:** none today.
- **Idempotency key:** `EventID`.

### `CardRemoved`

- **Emitted by:** `RemoveCard`.
- **Payload:**
  ```go
  type CardRemoved struct {
      DeckID    string
      CardID    string
      RemovedAt time.Time
  }
  ```
- **Consumers:** `practice.UserDeckProgress` projection (drops the matching
  `CardProgress`); `practice.Sessions` cleanup handler (drops the corresponding
  `RatedCard` from open sessions).
- **Idempotency key:** `CardID`.

### `CardsReordered`

- **Emitted by:** `ReorderCards`.
- **Payload:**
  ```go
  type CardsReordered struct {
      DeckID      string
      OrderedIDs  []string // new full ordering, length == card count
      ReorderedAt time.Time
  }
  ```
- **Consumers:** none today.
- **Idempotency key:** `EventID` (full reorder is replayable).

### `DeckGenerationRequested`

- **Emitted by:** `GenerateDeckWithAI` (entry point of the stub use case).
- **Payload:**
  ```go
  type DeckGenerationRequested struct {
      RequestID   string
      OwnerID     string
      Prompt      string // user's free-text seed
      RequestedAt time.Time
  }
  ```
- **Consumers:** none today (audit hook).
- **Idempotency key:** `RequestID` (= `Idempotency-Key` header value).

### `DeckGenerationCompleted`

- **Emitted by:** `GenerateDeckWithAI` (only when `AI_API_KEY` is configured; the stub
  emits this with an empty draft and `status = "not_implemented"` otherwise).
- **Payload:**
  ```go
  type DeckGenerationCompleted struct {
      RequestID   string
      OwnerID     string
      Status      string         // "ok" | "not_implemented" | "failed"
      Draft       AIDeckDraft    // zero-value when status != "ok"
      CompletedAt time.Time
  }
  type AIDeckDraft struct {
      Title string
      Cards []struct{ Term, Definition string }
  }
  ```
- **Consumers:** none today.
- **Idempotency key:** `RequestID`.

---

## practice

### `PracticeSessionStarted`

- **Emitted by:** `StartPracticeSession`.
- **Payload:**
  ```go
  type PracticeSessionStarted struct {
      SessionID string
      UserID    string
      DeckID    string
      Mode      string   // "tracked" | "untracked"
      CardIDs   []string // snapshot at start
      StartedAt time.Time
  }
  ```
- **Consumers:** none today.
- **Idempotency key:** `SessionID`.

### `CardRated`

- **Emitted by:** `RateCard`.
- **Payload:**
  ```go
  type CardRated struct {
      SessionID string
      UserID    string
      DeckID    string
      CardID    string
      Rating    int16   // 0=DontKnow, 1=StillLearning, 2=KnowKnow
      RatedAt   time.Time
  }
  ```
- **Consumers:** none today (the read-model is updated only on
  `PracticeSessionCompleted` for atomicity).
- **Idempotency key:** `(SessionID, CardID)` — re-rating overwrites prior payload.

### `PracticeSessionCompleted`

- **Emitted by:** `FinishPracticeSession`.
- **Payload:**
  ```go
  type PracticeSessionCompleted struct {
      SessionID    string
      UserID       string
      DeckID       string
      Mode         string // "tracked" | "untracked"
      CountDontKnow      int
      CountStillLearning int
      CountKnowKnow      int
      RatedCards   []struct {
          CardID string
          Rating int16
      }
      CompletedAt time.Time
  }
  ```
- **Consumers:** `practice.UserDeckProgress` projection — updates `CardProgress` for
  every entry in `RatedCards` when `Mode == "tracked"`.
- **Idempotency key:** `SessionID`.
