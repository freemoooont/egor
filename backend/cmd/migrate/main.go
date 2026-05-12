// Package main is the migrate binary entry point. It applies every
// *.up.sql under -dir/{shared,iam,decks,practice}/ to the Postgres DSN
// at -db, using the in-process runner that handles the .up/.down split
// goose v3.27.1 trips over.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/micocards/api/internal/infrastructure/postgres/migrate"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migrate: failed", slog.Any("err", err))
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	dir := flag.String("dir", "", "migrations root (containing shared/iam/decks/practice subdirs)")
	db := flag.String("db", "", "postgres DSN")
	flag.Parse()
	if *dir == "" || *db == "" {
		flag.Usage()
		return fmt.Errorf("both -dir and -db are required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *db)
	if err != nil {
		return fmt.Errorf("pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	files, err := migrate.LoadAll(*dir)
	if err != nil {
		return err
	}
	fmt.Printf("applying %d migrations from %s\n", len(files), *dir)
	for _, f := range files {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin %s: %w", f.Name, err)
		}
		if _, err := tx.Exec(ctx, f.Body); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("%s/%s: %w", f.Context, f.Name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", f.Name, err)
		}
		fmt.Printf("applied %s/%s\n", f.Context, f.Name)
	}
	return nil
}
