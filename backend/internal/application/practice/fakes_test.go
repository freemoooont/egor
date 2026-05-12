package practice_test

import (
	"context"
	"sync"

	domdecks "github.com/micocards/api/internal/domain/decks"
	dompractice "github.com/micocards/api/internal/domain/practice"
)

type fakeSessions struct {
	mu  sync.Mutex
	rec map[string]*dompractice.Session
}

func newFakeSessions() *fakeSessions { return &fakeSessions{rec: map[string]*dompractice.Session{}} }

func (f *fakeSessions) Save(_ context.Context, s *dompractice.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec[s.ID()] = s
	return nil
}

func (f *fakeSessions) ByID(_ context.Context, id string) (*dompractice.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.rec[id]
	if !ok {
		return nil, dompractice.ErrSessionNotFound
	}
	return s, nil
}

func (f *fakeSessions) LatestCompletedFor(_ context.Context, _, _ string) (*dompractice.Session, error) {
	return nil, dompractice.ErrSessionNotFound
}

type fakeProgress struct {
	mu sync.Mutex
	by map[string]*dompractice.UserDeckProgress // key = user|deck
}

func newFakeProgress() *fakeProgress {
	return &fakeProgress{by: map[string]*dompractice.UserDeckProgress{}}
}

func progKey(u, d string) string { return u + "|" + d }

func (f *fakeProgress) ByUserAndDeck(_ context.Context, u, d string) (*dompractice.UserDeckProgress, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.by[progKey(u, d)]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (f *fakeProgress) Save(_ context.Context, p *dompractice.UserDeckProgress) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.by[progKey(p.UserID, p.DeckID)] = p
	return nil
}

func (f *fakeProgress) DeleteByDeck(_ context.Context, deckID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, p := range f.by {
		if p.DeckID == deckID {
			delete(f.by, k)
		}
	}
	return nil
}

type fakeSnapshot struct {
	cards   map[string][]string
	owners  map[string]string
	missing map[string]bool
}

func (f fakeSnapshot) OwnerAndCards(_ context.Context, deckID string) (string, []string, error) {
	if f.missing[deckID] {
		return "", nil, domdecks.ErrDeckNotFound
	}
	return f.owners[deckID], f.cards[deckID], nil
}

type fakeUoW struct{}

func (fakeUoW) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type recordedEvent struct {
	Name string
	Ev   dompractice.Event
}

type fakeEvents struct {
	mu  sync.Mutex
	rec []recordedEvent
}

func (e *fakeEvents) Publish(_ context.Context, events ...dompractice.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ev := range events {
		e.rec = append(e.rec, recordedEvent{Name: ev.Name(), Ev: ev})
	}
	return nil
}

func (e *fakeEvents) Names() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, len(e.rec))
	for i, r := range e.rec {
		out[i] = r.Name
	}
	return out
}

func (e *fakeEvents) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rec = nil
}
