-- decks.cards — child entity inside the Deck aggregate.
-- Ordinal is 1-based and dense within a deck (Deck domain enforces this).
CREATE TABLE IF NOT EXISTS decks.cards (
    id           text        PRIMARY KEY,
    deck_id      text        NOT NULL REFERENCES decks.decks(id) ON DELETE CASCADE,
    ordinal      integer     NOT NULL CHECK (ordinal > 0),
    term         text        NOT NULL,
    definition   text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (deck_id, ordinal) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS cards_deck_idx ON decks.cards (deck_id, ordinal);
