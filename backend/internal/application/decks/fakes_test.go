package decks_test

import (
	"context"
	"sort"
	"sync"

	domdecks "github.com/micocards/api/internal/domain/decks"
)

type fakeDecks struct {
	mu  sync.Mutex
	rec map[string]*domdecks.Deck
	err error
}

func newFakeDecks() *fakeDecks { return &fakeDecks{rec: map[string]*domdecks.Deck{}} }

func (f *fakeDecks) Save(_ context.Context, d *domdecks.Deck) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	// Re-hydrate to take a snapshot of the current state.
	cards := d.Cards()
	var deletedAt *interface{}
	_ = deletedAt
	f.rec[d.ID()] = d
	_ = cards
	return nil
}

func (f *fakeDecks) ByID(_ context.Context, id string) (*domdecks.Deck, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.rec[id]
	if !ok {
		return nil, domdecks.ErrDeckNotFound
	}
	return d, nil
}

func (f *fakeDecks) ByOwner(_ context.Context, ownerID string, limit int, _ string) ([]*domdecks.Deck, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*domdecks.Deck{}
	for _, d := range f.rec {
		if d.OwnerID() == ownerID && !d.IsDeleted() {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, "", nil
}

func (f *fakeDecks) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rec, id)
	return nil
}

type fakeDecksUoW struct{}

func (fakeDecksUoW) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type recordedDeckEvent struct {
	Name string
	Ev   domdecks.Event
}

type fakeDeckEvents struct {
	mu  sync.Mutex
	rec []recordedDeckEvent
}

func (e *fakeDeckEvents) Publish(_ context.Context, events ...domdecks.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ev := range events {
		e.rec = append(e.rec, recordedDeckEvent{Name: ev.Name(), Ev: ev})
	}
	return nil
}

func (e *fakeDeckEvents) Names() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, len(e.rec))
	for i, r := range e.rec {
		out[i] = r.Name
	}
	return out
}

func (e *fakeDeckEvents) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rec = nil
}

type fakeAI struct {
	configured bool
	err        error
	draft      domdecks.AIDeckDraft
}

func (f fakeAI) IsConfigured() bool { return f.configured }
func (f fakeAI) Generate(_ context.Context, _ string) (domdecks.AIDeckDraft, error) {
	if f.err != nil {
		return domdecks.AIDeckDraft{}, f.err
	}
	return f.draft, nil
}
