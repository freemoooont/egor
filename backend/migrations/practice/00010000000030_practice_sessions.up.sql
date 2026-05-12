-- practice.sessions — Session aggregate root.
-- card_ids is the start-time deck snapshot. Status is one of:
--   'in_progress' | 'completed' | 'abandoned' (mirrors the domain enum).
CREATE TABLE IF NOT EXISTS practice.sessions (
    id            text        PRIMARY KEY,
    user_id       text        NOT NULL,
    deck_id       text        NOT NULL,
    mode          text        NOT NULL,
    status        text        NOT NULL,
    card_ids      jsonb       NOT NULL,
    started_at    timestamptz NOT NULL,
    completed_at  timestamptz NULL,
    abandoned_at  timestamptz NULL
);

CREATE INDEX IF NOT EXISTS sessions_user_deck_completed_idx
    ON practice.sessions (user_id, deck_id, completed_at DESC);
CREATE INDEX IF NOT EXISTS sessions_user_idx ON practice.sessions (user_id);
