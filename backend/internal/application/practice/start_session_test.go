package practice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppractice "github.com/micocards/api/internal/application/practice"
	dompractice "github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
)

const (
	user = "u-1"
	deck = "d-1"
)

func TestStartSession_HappyPath(t *testing.T) {
	sess := newFakeSessions()
	events := &fakeEvents{}
	uc := apppractice.StartSession{
		Sessions:  sess,
		Snapshots: fakeSnapshot{owners: map[string]string{deck: user}, cards: map[string][]string{deck: {"c-1", "c-2"}}},
		IDs:       idgen.NewSequential("s"),
		Clock:     clock.NewFixed(time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)),
		UoW:       fakeUoW{},
		Events:    events,
	}
	out, err := uc.Handle(context.Background(), apppractice.StartSessionInput{UserID: user, DeckID: deck, Mode: "tracked"})
	require.NoError(t, err)
	require.NotEmpty(t, out.SessionID)
	require.Equal(t, []string{"c-1", "c-2"}, out.CardIDs)
	require.Equal(t, []string{"practice.PracticeSessionStarted"}, events.Names())
	require.Len(t, sess.rec, 1)
}

func TestStartSession_ForbiddenWhenWrongOwner(t *testing.T) {
	uc := apppractice.StartSession{
		Sessions:  newFakeSessions(),
		Snapshots: fakeSnapshot{owners: map[string]string{deck: "other"}, cards: map[string][]string{deck: {"c-1"}}},
		IDs:       idgen.NewSequential("s"),
		Clock:     clock.NewFixed(time.Now().UTC()),
		UoW:       fakeUoW{},
		Events:    &fakeEvents{},
	}
	_, err := uc.Handle(context.Background(), apppractice.StartSessionInput{UserID: user, DeckID: deck, Mode: "tracked"})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}

func TestStartSession_DeckNotFound(t *testing.T) {
	uc := apppractice.StartSession{
		Sessions:  newFakeSessions(),
		Snapshots: fakeSnapshot{missing: map[string]bool{deck: true}},
		IDs:       idgen.NewSequential("s"),
		Clock:     clock.NewFixed(time.Now().UTC()),
		UoW:       fakeUoW{},
		Events:    &fakeEvents{},
	}
	_, err := uc.Handle(context.Background(), apppractice.StartSessionInput{UserID: user, DeckID: deck, Mode: "tracked"})
	require.Error(t, err)
}

func TestStartSession_RejectsInvalidMode(t *testing.T) {
	uc := apppractice.StartSession{
		Sessions:  newFakeSessions(),
		Snapshots: fakeSnapshot{owners: map[string]string{deck: user}, cards: map[string][]string{deck: {"c-1"}}},
		IDs:       idgen.NewSequential("s"),
		Clock:     clock.NewFixed(time.Now().UTC()),
		UoW:       fakeUoW{},
		Events:    &fakeEvents{},
	}
	_, err := uc.Handle(context.Background(), apppractice.StartSessionInput{UserID: user, DeckID: deck, Mode: "invalid"})
	require.ErrorIs(t, err, dompractice.ErrInvalidPracticeMode)
}

func TestStartSession_RejectsEmptyDeck(t *testing.T) {
	uc := apppractice.StartSession{
		Sessions:  newFakeSessions(),
		Snapshots: fakeSnapshot{owners: map[string]string{deck: user}, cards: map[string][]string{deck: {}}},
		IDs:       idgen.NewSequential("s"),
		Clock:     clock.NewFixed(time.Now().UTC()),
		UoW:       fakeUoW{},
		Events:    &fakeEvents{},
	}
	_, err := uc.Handle(context.Background(), apppractice.StartSessionInput{UserID: user, DeckID: deck, Mode: "tracked"})
	require.ErrorIs(t, err, dompractice.ErrDeckEmpty)
}

func TestStartSession_ForbiddenWhenUserEmpty(t *testing.T) {
	uc := apppractice.StartSession{IDs: idgen.NewSequential("s"), Clock: clock.NewFixed(time.Now().UTC())}
	_, err := uc.Handle(context.Background(), apppractice.StartSessionInput{Mode: "tracked"})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}
