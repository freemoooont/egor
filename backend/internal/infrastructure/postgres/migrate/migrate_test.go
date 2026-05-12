package migrate

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// migrationsRoot computes backend/migrations from this test file's location.
func migrationsRoot(t *testing.T) string {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", "..", "..", "migrations"))
}

func TestLoadAll_FindsExpectedFiles(t *testing.T) {
	files, err := LoadAll(migrationsRoot(t))
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one migration file")
	}
	wantContexts := []string{"shared", "iam", "decks", "practice"}
	have := map[string]bool{}
	for _, f := range files {
		have[f.Context] = true
		if !strings.HasSuffix(f.Name, ".up.sql") {
			t.Fatalf("unexpected non-up file: %s", f.Name)
		}
	}
	for _, w := range wantContexts {
		if !have[w] {
			t.Fatalf("missing migrations for context %s", w)
		}
	}
}

func TestLoadAll_PreservesAlphabeticOrder(t *testing.T) {
	files, err := LoadAll(migrationsRoot(t))
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	// Within a single context, names should be ASC.
	prev := map[string]string{}
	for _, f := range files {
		if last, ok := prev[f.Context]; ok {
			if f.Name <= last {
				t.Fatalf("context %s: %s did not sort after %s", f.Context, f.Name, last)
			}
		}
		prev[f.Context] = f.Name
	}
}
