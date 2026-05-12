-- iam.refresh_tokens — rotating refresh-token family chain (ADR 0003).
-- We never store plaintext; only the sha256 hash of the opaque value.
CREATE TABLE IF NOT EXISTS iam.refresh_tokens (
    id            text        PRIMARY KEY,
    family_id     text        NOT NULL,
    user_id       text        NOT NULL,
    parent_id     text        NOT NULL DEFAULT '',
    opaque_hash   text        NOT NULL UNIQUE,
    issued_at     timestamptz NOT NULL,
    expires_at    timestamptz NOT NULL,
    revoked_at    timestamptz NULL,
    revoke_note   text        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS refresh_tokens_family_idx ON iam.refresh_tokens (family_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_user_idx   ON iam.refresh_tokens (user_id);
