-- practice.user_deck_progress — per-user per-deck read model (UserDeckProgress).
-- The aggregate stores a slice of CardProgress; we serialize the whole list as
-- JSONB to keep the read model self-contained in one row.
CREATE TABLE IF NOT EXISTS practice.user_deck_progress (
    user_id            text        NOT NULL,
    deck_id            text        NOT NULL,
    cards              jsonb       NOT NULL DEFAULT '[]'::jsonb,
    know_count         integer     NOT NULL DEFAULT 0,
    learning_count     integer     NOT NULL DEFAULT 0,
    dont_know_count    integer     NOT NULL DEFAULT 0,
    last_practiced_at  timestamptz NOT NULL,
    updated_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, deck_id)
);

CREATE INDEX IF NOT EXISTS user_deck_progress_deck_idx
    ON practice.user_deck_progress (deck_id);
