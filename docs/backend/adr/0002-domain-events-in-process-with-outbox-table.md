# ADR 0002 — Domain events: in-process synchronous dispatch + transactional outbox

- **Status:** accepted
- **Date:** 2026-05-07
- **Deciders:** backend agent (TASK_ID `micocards-mvp`)

## Context

Domain events (`domain-events.md`) cross context boundaries — `practice` reacts to
`DeckDeleted` and `CardRemoved` from `decks`, and the `UserDeckProgress` projection
reacts to `PracticeSessionCompleted`. We need a delivery story that:

- Survives process crashes — an event whose write was committed must eventually be
  delivered, even if the consumer or the publisher restarts mid-handoff.
- Does not require any external broker for v1 (Postgres is the only datastore per
  the spec — no Redis, no Kafka, no NATS).
- Allows us to flip to async transport later without touching domain or use-case code.
- Keeps the transactional contract clean: an event is "real" only if its emitting
  aggregate was committed.

## Decision

**Synchronous in-process dispatch via a thin event bus, _and_ persisted to a
transactional outbox in the same `pgx.Tx` that produced the event.**

Mechanics:

1. Each context has its own `outbox` table (`iam.outbox`, `decks.outbox`,
   `practice.outbox`) with columns:
   `event_id uuid PK, event_name text, occurred_at timestamptz, aggregate_id text,
    payload jsonb, dispatched_at timestamptz null, dispatch_attempts int default 0`.
2. Use cases call `events.Publish(ctx, evt)` inside their `pgx.Tx`. The publisher
   inserts the row into the outbox and, on tx commit, dispatches synchronously to
   in-process subscribers. On dispatch success it sets `dispatched_at = now()`.
3. Subscribers register at `cmd/api/main.go` composition root. They are pure
   functions taking `(context.Context, EventEnvelope) error`.
4. A small janitor goroutine periodically scans `outbox` rows where
   `dispatched_at IS NULL AND occurred_at < now() - 30s` and re-dispatches them.
   This catches the (rare) "tx committed but in-process dispatcher panicked" case.
5. Subscribers are written to be idempotent — they use the event's idempotency key
   (see `domain-events.md`) to deduplicate replays. The `UserDeckProgress` projection
   is the one consumer that genuinely needs this.

The event bus interface is small enough that swapping it for a Kafka/NATS publisher
later is a single-package change.

## Alternatives considered

- **Pure in-memory dispatch, no outbox.** Lost events on crash. Rejected.
- **Pure outbox + worker pull (no in-process sync).** Forces every domain reaction to
  pay the worker-poll latency, and complicates testing. Rejected for v1.
- **Postgres `LISTEN/NOTIFY`-based bus.** Workable, but ties us to a single-process
  dispatcher topology and requires a long-lived connection per consumer. We can add
  it later as a transport on top of the outbox without changing the domain.
- **External broker (Kafka/NATS/RabbitMQ).** Out of scope for v1; spec restricts the
  stack to Postgres only.

## Consequences

**Positive.**
- Atomicity: an event is durable iff its aggregate write committed. No "phantom
  events" or "lost events".
- Latency: subscribers run in the same request, so the user sees the projection
  update immediately on `FinishPracticeSession`.
- Easy migration path: flip in-process dispatch to async by replacing the dispatcher
  implementation; the domain and use cases stay untouched.

**Negative.**
- Subscribers run inside the request — a slow subscriber slows the response. We
  budget < 5 ms per subscriber and put expensive work behind a future async worker.
- Two write paths to keep correct: the event bus on commit, and the janitor on
  recovery. Both go through the same `dispatch(envelope)` function, so the divergence
  is bounded to one place.

**Neutral.**
- The `outbox` table grows. We add a periodic vacuum (rows older than 30 days with
  `dispatched_at != NULL` are deleted) — same janitor goroutine.
- Tests assert both: the aggregate state AND the outbox row. This is a deliberate
  test-pyramid choice — the outbox is part of the use case's observable behaviour.
