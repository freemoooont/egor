package practice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppractice "github.com/micocards/api/internal/application/practice"
	dompractice "github.com/micocards/api/internal/domain/practice"
	"github.com/micocards/api/internal/infrastructure/clock"
)

func seedSession(t *testing.T, sess *fakeSessions, mode dompractice.SessionMode) *dompractice.Session {
	t.Helper()
	s, err := dompractice.NewSession("s-1", user, deck, mode, []string{"c-1", "c-2"}, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, sess.Save(context.Background(), s))
	return s
}

func TestRateCard_HappyPath(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	events := &fakeEvents{}
	uc := apppractice.RateCard{Sessions: sess, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeUoW{}, Events: events}
	out, err := uc.Handle(context.Background(), apppractice.RateCardInput{UserID: user, SessionID: "s-1", CardID: "c-1", Rating: 2})
	require.NoError(t, err)
	require.Equal(t, int16(2), out.Rating)
	require.Equal(t, []string{"practice.CardRated"}, events.Names())
}

func TestRateCard_ForbiddenWhenStranger(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	uc := apppractice.RateCard{Sessions: sess, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeUoW{}, Events: &fakeEvents{}}
	_, err := uc.Handle(context.Background(), apppractice.RateCardInput{UserID: "other", SessionID: "s-1", CardID: "c-1", Rating: 0})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}

func TestRateCard_RejectsInvalidRating(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	uc := apppractice.RateCard{Sessions: sess, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeUoW{}, Events: &fakeEvents{}}
	_, err := uc.Handle(context.Background(), apppractice.RateCardInput{UserID: user, SessionID: "s-1", CardID: "c-1", Rating: 99})
	require.ErrorIs(t, err, dompractice.ErrInvalidRating)
}

func TestRateCard_RejectsCardNotInSnapshot(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	uc := apppractice.RateCard{Sessions: sess, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeUoW{}, Events: &fakeEvents{}}
	_, err := uc.Handle(context.Background(), apppractice.RateCardInput{UserID: user, SessionID: "s-1", CardID: "c-XYZ", Rating: 0})
	require.ErrorIs(t, err, dompractice.ErrCardNotInSession)
}

func TestRateCard_UntrackedRejected(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeUntracked)
	uc := apppractice.RateCard{Sessions: sess, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeUoW{}, Events: &fakeEvents{}}
	_, err := uc.Handle(context.Background(), apppractice.RateCardInput{UserID: user, SessionID: "s-1", CardID: "c-1", Rating: 0})
	require.ErrorIs(t, err, dompractice.ErrSessionUntracked)
}

func TestRateCard_ForbiddenWhenUserEmpty(t *testing.T) {
	uc := apppractice.RateCard{}
	_, err := uc.Handle(context.Background(), apppractice.RateCardInput{Rating: 0})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}

func TestRateCard_SessionNotFound(t *testing.T) {
	uc := apppractice.RateCard{Sessions: newFakeSessions(), Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeUoW{}, Events: &fakeEvents{}}
	_, err := uc.Handle(context.Background(), apppractice.RateCardInput{UserID: user, SessionID: "missing", CardID: "c-1", Rating: 0})
	require.ErrorIs(t, err, dompractice.ErrSessionNotFound)
}
