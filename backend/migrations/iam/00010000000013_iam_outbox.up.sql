-- iam.outbox — transactional outbox for the iam context (ADR 0002).
CREATE TABLE IF NOT EXISTS iam.outbox (
    id                  bigserial   PRIMARY KEY,
    aggregate_type      text        NOT NULL DEFAULT 'iam',
    aggregate_id        text        NOT NULL DEFAULT '',
    event_name          text        NOT NULL,
    payload             jsonb       NOT NULL,
    idempotency_key     text        NOT NULL DEFAULT '',
    created_at          timestamptz NOT NULL DEFAULT now(),
    dispatched_at       timestamptz NULL,
    dispatch_attempts   integer     NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS iam_outbox_undispatched_idx
    ON iam.outbox (created_at) WHERE dispatched_at IS NULL;
