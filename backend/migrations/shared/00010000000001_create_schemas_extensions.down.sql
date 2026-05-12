-- Down for the cross-context bootstrap migration.
DROP SCHEMA IF EXISTS practice CASCADE;
DROP SCHEMA IF EXISTS decks CASCADE;
DROP SCHEMA IF EXISTS iam CASCADE;
-- pgcrypto is left installed (extensions are cheap and shared).
