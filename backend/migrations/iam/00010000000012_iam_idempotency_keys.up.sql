-- iam.idempotency_keys — cross-cutting (ADR 0005).
-- Layer 1's IdempotencyEntry exposes scope/key/request_hash/response_status/
-- response_body/expires_at. We persist exactly that, with a composite primary
-- key on (scope, key).
CREATE TABLE IF NOT EXISTS iam.idempotency_keys (
    scope            text        NOT NULL,
    key              text        NOT NULL,
    request_hash     text        NOT NULL,
    response_status  integer     NOT NULL,
    response_body    bytea       NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    expires_at       timestamptz NOT NULL,
    PRIMARY KEY (scope, key)
);

CREATE INDEX IF NOT EXISTS idempotency_keys_expires_idx ON iam.idempotency_keys (expires_at);
