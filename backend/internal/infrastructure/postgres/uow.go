// Package postgres provides the cross-repo plumbing shared by every context's
// repo subpackage: a connection pool, a UnitOfWork that opens a pgx.Tx and
// stores it on context, and helpers that pull the active tx out of the
// context (or fall back to the pool).
//
// Per docs/backend/CLAUDE.md "pgx/v5 + sqlc + goose conventions": one pgx.Tx
// per use case, opened by the UnitOfWork port. Repositories pull the active tx
// out of context.Context — they never receive a connection directly.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// txKey is the unexported context-key type for storing the active pgx.Tx.
type txKey struct{}

// withTx returns a copy of ctx that carries tx.
func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// txFrom returns the active tx on ctx, or nil if there is none.
func txFrom(ctx context.Context) pgx.Tx {
	v := ctx.Value(txKey{})
	if v == nil {
		return nil
	}
	tx, _ := v.(pgx.Tx)
	return tx
}

// Querier is the small interface every repository method calls. Both
// pgx.Tx and *pgxpool.Pool implement Exec/Query/QueryRow with these
// signatures, so the wrapper code below is direct.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconnCommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// pgconnCommandTag is the subset of pgconn.CommandTag we exercise.
type pgconnCommandTag interface {
	RowsAffected() int64
	String() string
}

// Conn returns the pgx-compatible handle for the active tx if there is one,
// otherwise the supplied pool. Repository methods call this on every query.
func Conn(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx := txFrom(ctx); tx != nil {
		return txWrapper{tx: tx}
	}
	return poolWrapper{pool: pool}
}

type txWrapper struct{ tx pgx.Tx }

func (w txWrapper) Exec(ctx context.Context, sql string, args ...any) (pgconnCommandTag, error) {
	t, err := w.tx.Exec(ctx, sql, args...)
	return t, err
}
func (w txWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return w.tx.Query(ctx, sql, args...)
}
func (w txWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return w.tx.QueryRow(ctx, sql, args...)
}

type poolWrapper struct{ pool *pgxpool.Pool }

func (w poolWrapper) Exec(ctx context.Context, sql string, args ...any) (pgconnCommandTag, error) {
	t, err := w.pool.Exec(ctx, sql, args...)
	return t, err
}
func (w poolWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return w.pool.Query(ctx, sql, args...)
}
func (w poolWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return w.pool.QueryRow(ctx, sql, args...)
}

// UnitOfWork is the concrete implementation of the application-layer UnitOfWork
// port (one per context). Use cases call Do(ctx, fn); the impl opens a tx,
// places it on the context, runs fn, and commits/rolls back.
//
// A nested Do reuses the existing tx instead of opening a new one — this lets
// composite use cases (e.g. middleware that already opened a tx for
// idempotency) share the same transaction with their inner use cases.
type UnitOfWork struct {
	pool *pgxpool.Pool
}

// NewUnitOfWork builds a UoW backed by the supplied pgxpool.Pool.
func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork { return &UnitOfWork{pool: pool} }

// Do runs fn inside a pgx.Tx. If ctx already carries a tx, fn is called with
// the same tx (no nesting, no savepoints) — the outer caller owns commit/abort.
func (u *UnitOfWork) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if tx := txFrom(ctx); tx != nil {
		return fn(ctx)
	}
	if u.pool == nil {
		return errors.New("postgres.UnitOfWork: pool is nil")
	}
	tx, err := u.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	ctxTx := withTx(ctx, tx)
	if err := fn(ctxTx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
