// Package migrate is a small in-process goose-compatible migration runner.
//
// It exists so integration tests (and main, when goose isn't installed) can
// apply migrations against a fresh DB without depending on the goose binary.
// Filenames follow the goose convention `<unix-ts>_<name>.up.sql`/`.down.sql`
// so the same files work with `goose -dir backend/migrations/<context> postgres
// $URL up` once the binary is installed.
package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationFile is a single .up.sql we plan to apply.
type MigrationFile struct {
	Path    string
	Name    string
	Body    string
	Context string // "shared" | "iam" | "decks" | "practice"
}

// orderedContexts is the canonical apply order: bootstrap shared first, then
// each context. Mirrors `make migrate-up` in docs/backend/CLAUDE.md.
var orderedContexts = []string{"shared", "iam", "decks", "practice"}

// LoadAll reads every *.up.sql under dir/<context>/ for each context in
// orderedContexts. Files are returned sorted ASC by basename within a context.
func LoadAll(rootDir string) ([]MigrationFile, error) {
	var out []MigrationFile
	for _, ctx := range orderedContexts {
		dir := filepath.Join(rootDir, ctx)
		ents, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("migrate: read %s: %w", dir, err)
		}
		var names []string
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			n := e.Name()
			if strings.HasSuffix(n, ".up.sql") {
				names = append(names, n)
			}
		}
		sort.Strings(names)
		for _, n := range names {
			full := filepath.Join(dir, n)
			body, err := os.ReadFile(full)
			if err != nil {
				return nil, fmt.Errorf("migrate: read %s: %w", full, err)
			}
			out = append(out, MigrationFile{
				Path: full, Name: n, Body: string(body), Context: ctx,
			})
		}
	}
	return out, nil
}

// ApplyAll executes every migration body against the pool. Files are applied
// inside a single tx per file (so a single failure rolls back its own file
// without poisoning the rest). Idempotent SQL (CREATE ... IF NOT EXISTS) makes
// the runner safe against reruns.
func ApplyAll(ctx context.Context, pool *pgxpool.Pool, rootDir string) error {
	files, err := LoadAll(rootDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := apply(ctx, pool, f); err != nil {
			return fmt.Errorf("migrate: %s: %w", f.Name, err)
		}
	}
	return nil
}

func apply(ctx context.Context, pool *pgxpool.Pool, f MigrationFile) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, f.Body); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
