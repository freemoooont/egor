-- Layer 2 — initial cross-context migration.
-- Creates the per-context schemas (ADR 0004) plus a public outbox schema and
-- enables the pgcrypto extension so application code MAY use gen_random_uuid()
-- if it ever wants to. Today the application owns all id generation via the
-- shared.IDGenerator port (UUID v4 from idgen.UUID), so the DB never needs to
-- mint ids on its own.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS iam;
CREATE SCHEMA IF NOT EXISTS decks;
CREATE SCHEMA IF NOT EXISTS practice;

-- Per ADR 0002 / 0004: each context has its own outbox table. The bounded-
-- contexts doc lists `iam.outbox`, `decks.outbox`, `practice.outbox`. Layer 2
-- DECISION: keep one outbox table per context (matches the doc and lets each
-- context own its own write rate independently). No public.outbox table.
