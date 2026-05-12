-- practice.session_card_ratings (a.k.a. rated_cards) — child entity table for
-- a Session. The deliverables list spells the table `session_card_ratings`;
-- the bounded-contexts doc spells it `rated_cards`. We follow the deliverables
-- list (matches how DDL queries below name it) but keep a comment in case the
-- doc is the source of truth in a later sprint.
CREATE TABLE IF NOT EXISTS practice.session_card_ratings (
    id          text        PRIMARY KEY,
    session_id  text        NOT NULL REFERENCES practice.sessions(id) ON DELETE CASCADE,
    card_id     text        NOT NULL,
    rating      smallint    NOT NULL CHECK (rating IN (0, 1, 2)),
    rated_at    timestamptz NOT NULL,
    UNIQUE (session_id, card_id)
);

CREATE INDEX IF NOT EXISTS session_card_ratings_session_idx
    ON practice.session_card_ratings (session_id);
