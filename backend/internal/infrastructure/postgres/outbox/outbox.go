// Package outbox implements the per-context transactional outbox (ADR 0002).
// One Outbox value per context (iam, decks, practice). Each instance writes
// to its context's <schema>.outbox table inside the active pgx.Tx, so events
// are durable iff the use-case tx commits.
package outbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	pg "github.com/micocards/api/internal/infrastructure/postgres"
)

// Outbox is the per-context append-only outbox repository. The schema name is
// fixed at construction; do not mutate it.
type Outbox struct {
	pool   *pgxpool.Pool
	schema string // "iam" | "decks" | "practice"
}

// New builds an Outbox for the given schema.
func New(pool *pgxpool.Pool, schema string) (*Outbox, error) {
	switch schema {
	case "iam", "decks", "practice":
	default:
		return nil, errors.New("outbox: unsupported schema: " + schema)
	}
	return &Outbox{pool: pool, schema: schema}, nil
}

// Append writes one outbox row inside the active tx (or directly against the
// pool if no tx is active — for cross-context smoke tests). It implements the
// application-layer Outbox port. The aggregate_type column is set to the
// schema name for v1; finer-grained aggregate types can come later.
func (o *Outbox) Append(ctx context.Context, eventName string, payload []byte, idempotencyKey string) error {
	if eventName == "" {
		return errors.New("outbox: event name is required")
	}
	q := pg.Conn(ctx, o.pool)
	sql := fmt.Sprintf(`
INSERT INTO %s.outbox (aggregate_type, aggregate_id, event_name, payload, idempotency_key)
VALUES ($1, $2, $3, $4::jsonb, $5)
`, o.schema)
	if _, err := q.Exec(ctx, sql, o.schema, "", eventName, string(payload), idempotencyKey); err != nil {
		return fmt.Errorf("outbox.Append %s: %w", o.schema, err)
	}
	return nil
}

// Row mirrors a single outbox row (for the dispatcher and tests).
type Row struct {
	ID             int64
	AggregateType  string
	AggregateID    string
	EventName      string
	Payload        []byte
	IdempotencyKey string
}

// FetchUndispatchedBatch returns up to limit undispatched rows ordered by id
// asc. The dispatcher calls this and forwards to in-process subscribers.
func (o *Outbox) FetchUndispatchedBatch(ctx context.Context, limit int) ([]Row, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := pg.Conn(ctx, o.pool).Query(ctx, fmt.Sprintf(`
SELECT id, aggregate_type, aggregate_id, event_name, payload, idempotency_key
FROM %s.outbox
WHERE dispatched_at IS NULL
ORDER BY id ASC
LIMIT $1
`, o.schema), limit)
	if err != nil {
		return nil, fmt.Errorf("outbox.FetchUndispatchedBatch %s: %w", o.schema, err)
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		var raw []byte
		if err := rows.Scan(&r.ID, &r.AggregateType, &r.AggregateID, &r.EventName, &raw, &r.IdempotencyKey); err != nil {
			return nil, err
		}
		r.Payload = raw
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// MarkDispatched stamps dispatched_at = now() on the given id.
func (o *Outbox) MarkDispatched(ctx context.Context, id int64) error {
	_, err := pg.Conn(ctx, o.pool).Exec(ctx, fmt.Sprintf(`
UPDATE %s.outbox
SET dispatched_at = now(), dispatch_attempts = dispatch_attempts + 1
WHERE id = $1
`, o.schema), id)
	if err != nil {
		return fmt.Errorf("outbox.MarkDispatched %s: %w", o.schema, err)
	}
	return nil
}
