//go:build integration

package outbox_test

import (
	"context"
	"testing"

	pg "github.com/micocards/api/internal/infrastructure/postgres"
	"github.com/micocards/api/internal/infrastructure/postgres/outbox"
	"github.com/micocards/api/internal/infrastructure/postgres/testdb"
)

type fakeForwarder struct {
	calls []struct {
		name           string
		payload        []byte
		idempotencyKey string
	}
	failOn string
	errOn  error
}

func (f *fakeForwarder) Forward(_ context.Context, name string, payload []byte, idem string) error {
	if f.failOn != "" && name == f.failOn {
		return f.errOn
	}
	f.calls = append(f.calls, struct {
		name           string
		payload        []byte
		idempotencyKey string
	}{name, payload, idem})
	return nil
}

func TestOutbox_AppendThenDispatch(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	box, err := outbox.New(pool, "iam")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	uow := pg.NewUnitOfWork(pool)
	if err := uow.Do(ctx, func(ctx context.Context) error {
		if err := box.Append(ctx, "iam.UserRegistered", []byte(`{"id":"u-1"}`), ""); err != nil {
			return err
		}
		return box.Append(ctx, "iam.UserLoggedIn", []byte(`{"id":"u-1"}`), "k-1")
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	rows, err := box.FetchUndispatchedBatch(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUndispatchedBatch: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0].EventName != "iam.UserRegistered" {
		t.Fatalf("first event: %s", rows[0].EventName)
	}

	// Dispatch.
	fwd := &fakeForwarder{}
	d := outbox.NewDispatcher(box, fwd, 0)
	n, err := d.DispatchOnce(ctx)
	if err != nil {
		t.Fatalf("DispatchOnce: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 dispatched, got %d", n)
	}
	if len(fwd.calls) != 2 {
		t.Fatalf("want 2 forwarder calls, got %d", len(fwd.calls))
	}
	leftovers, _ := box.FetchUndispatchedBatch(ctx, 10)
	if len(leftovers) != 0 {
		t.Fatalf("want 0 undispatched, got %d", len(leftovers))
	}
}

func TestOutbox_AppendOnAllSchemas(t *testing.T) {
	pool, ctx := testdb.New(t)
	defer testdb.CleanAll(t, ctx, pool)
	for _, schema := range []string{"iam", "decks", "practice"} {
		box, err := outbox.New(pool, schema)
		if err != nil {
			t.Fatalf("New %s: %v", schema, err)
		}
		if err := box.Append(ctx, schema+".Smoke", []byte(`{}`), ""); err != nil {
			t.Fatalf("Append %s: %v", schema, err)
		}
		rows, _ := box.FetchUndispatchedBatch(ctx, 10)
		if len(rows) != 1 {
			t.Fatalf("%s: want 1 row, got %d", schema, len(rows))
		}
	}
}
