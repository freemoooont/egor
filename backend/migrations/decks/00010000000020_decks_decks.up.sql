-- decks.decks — deck aggregate root.
-- Soft-delete via deleted_at; queries use it as a filter at the repo level.
CREATE TABLE IF NOT EXISTS decks.decks (
    id          text        PRIMARY KEY,
    owner_id    text        NOT NULL,
    title       text        NOT NULL,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz NULL
);

CREATE INDEX IF NOT EXISTS decks_owner_idx ON decks.decks (owner_id) WHERE deleted_at IS NULL;
