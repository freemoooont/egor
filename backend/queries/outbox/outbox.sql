-- name: AppendOutbox :exec
-- The schema is interpolated by the caller (one outbox table per context).
-- This file is documentation-only: hand-written repos in the outbox package
-- inline the per-schema SQL because sqlc cannot parameterise schema names.

-- name: FetchUndispatched :many
-- (See above — schema is set by the calling repository.)

-- name: MarkDispatched :exec
-- (See above — schema is set by the calling repository.)
