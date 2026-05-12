package events

import (
	"context"
	"errors"
	"testing"
)

type fakeEv struct{ name string }

func (f fakeEv) Name() string { return f.name }

func TestPublisher_FanOut(t *testing.T) {
	p := NewPublisher()
	var calls []string
	p.Subscribe("a", func(_ context.Context, ev Event) error {
		calls = append(calls, "a1:"+ev.Name())
		return nil
	})
	p.Subscribe("a", func(_ context.Context, ev Event) error {
		calls = append(calls, "a2:"+ev.Name())
		return nil
	})
	p.Subscribe("b", func(_ context.Context, ev Event) error {
		calls = append(calls, "b:"+ev.Name())
		return nil
	})
	if err := p.Publish(context.Background(), fakeEv{"a"}, fakeEv{"b"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	want := []string{"a1:a", "a2:a", "b:b"}
	if len(calls) != 3 {
		t.Fatalf("want 3 calls, got %v", calls)
	}
	for i, c := range calls {
		if c != want[i] {
			t.Fatalf("calls[%d]: want %s, got %s", i, want[i], c)
		}
	}
}

var errBoom = errors.New("boom")

func TestPublisher_StopsOnError(t *testing.T) {
	p := NewPublisher()
	var ran bool
	p.Subscribe("a", func(_ context.Context, _ Event) error { return errBoom })
	p.Subscribe("a", func(_ context.Context, _ Event) error {
		ran = true
		return nil
	})
	if err := p.Publish(context.Background(), fakeEv{"a"}); !errors.Is(err, errBoom) {
		t.Fatalf("want errBoom, got %v", err)
	}
	if ran {
		t.Fatal("second handler should not run after error")
	}
}
