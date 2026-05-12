//go:build integration

// Package testdb spins up a Postgres testcontainer, runs the project's
// goose-style migrations against it, and hands back a *pgxpool.Pool plus a
// cleanup func. Tests build with `-tags=integration` to avoid pulling docker
// into the unit-test path.
package testdb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/micocards/api/internal/infrastructure/postgres/migrate"
)

// MigrationsRoot returns the absolute path to backend/migrations regardless of
// which package the calling _test.go lives in.
func MigrationsRoot(t *testing.T) string {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	// here = .../backend/internal/infrastructure/postgres/testdb/postgres.go
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", "..", "..", "migrations"))
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("testdb: migrations root not found: %v", err)
	}
	return root
}

// New starts a Postgres testcontainer, applies all migrations, and returns a
// pool + cleanup. INTEGRATION=0 skips immediately. The container TTL is bound
// to the test so the cleanup must always run via t.Cleanup.
func New(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	if os.Getenv("INTEGRATION") == "0" {
		t.Skip("INTEGRATION=0 — skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("micocards"),
		tcpostgres.WithUsername("micocards"),
		tcpostgres.WithPassword("micocards"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("testdb: start container: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, c := context.WithTimeout(context.Background(), 30*time.Second)
		defer c()
		_ = container.Terminate(stopCtx)
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("testdb: dsn: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("testdb: connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := migrate.ApplyAll(ctx, pool, MigrationsRoot(t)); err != nil {
		t.Fatalf("testdb: migrate: %v", err)
	}
	return pool, ctx
}

// MustExec runs sql in ctx with no result and fails the test on error. Handy
// for per-test setup.
func MustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("testdb.MustExec: %v\nsql: %s", err, sql)
	}
}

// CleanAll truncates every owned table so the next test starts on a blank DB.
// We TRUNCATE rather than DROP/recreate because it's ~10× faster.
func CleanAll(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	tables := []string{
		"iam.idempotency_keys",
		"iam.refresh_tokens",
		"iam.users",
		"iam.outbox",
		"decks.cards",
		"decks.decks",
		"decks.outbox",
		"practice.session_card_ratings",
		"practice.sessions",
		"practice.user_deck_progress",
		"practice.outbox",
	}
	for _, tbl := range tables {
		if _, err := pool.Exec(ctx, fmt.Sprintf(`TRUNCATE TABLE %s RESTART IDENTITY CASCADE`, tbl)); err != nil {
			t.Fatalf("testdb.CleanAll %s: %v", tbl, err)
		}
	}
}
