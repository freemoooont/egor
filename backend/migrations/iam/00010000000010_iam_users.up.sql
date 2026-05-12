-- iam.users — auth identity owner.
-- Layer 1 invariants: User_EmailAddressIsLowerCasedOnConstruction guarantees
-- the value is already lower-cased before it lands here, so a UNIQUE on the raw
-- column is enough — no functional index needed.
CREATE TABLE IF NOT EXISTS iam.users (
    id              text        PRIMARY KEY,
    email           text        NOT NULL UNIQUE,
    password_hash   text        NOT NULL,
    display_name    text        NOT NULL,
    avatar_kind     text        NOT NULL DEFAULT 'none',
    avatar_ref      text        NOT NULL DEFAULT '',
    registered_at   timestamptz NOT NULL,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS users_email_idx ON iam.users (email);
