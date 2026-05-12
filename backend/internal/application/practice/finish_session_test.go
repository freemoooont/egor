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

func TestFinishSession_HappyPath(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	events := &fakeEvents{}
	clk := clock.NewFixed(time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC))
	rate := apppractice.RateCard{Sessions: sess, Clock: clk, UoW: fakeUoW{}, Events: events}
	_, err := rate.Handle(context.Background(), apppractice.RateCardInput{UserID: user, SessionID: "s-1", CardID: "c-1", Rating: 2})
	require.NoError(t, err)
	events.Reset()

	uc := apppractice.FinishSession{Sessions: sess, Clock: clk, UoW: fakeUoW{}, Events: events}
	out, err := uc.Handle(context.Background(), apppractice.FinishSessionInput{UserID: user, SessionID: "s-1"})
	require.NoError(t, err)
	require.Equal(t, 1, out.CountKnowKnow)
	require.Equal(t, []string{"practice.PracticeSessionCompleted"}, events.Names())
}

func TestFinishSession_Idempotent(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	events := &fakeEvents{}
	clk := clock.NewFixed(time.Now().UTC())
	uc := apppractice.FinishSession{Sessions: sess, Clock: clk, UoW: fakeUoW{}, Events: events}
	out1, err := uc.Handle(context.Background(), apppractice.FinishSessionInput{UserID: user, SessionID: "s-1"})
	require.NoError(t, err)
	out2, err := uc.Handle(context.Background(), apppractice.FinishSessionInput{UserID: user, SessionID: "s-1"})
	require.NoError(t, err)
	require.Equal(t, out1.CompletedAt, out2.CompletedAt)
}

func TestFinishSession_ForbiddenStranger(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	uc := apppractice.FinishSession{Sessions: sess, Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeUoW{}, Events: &fakeEvents{}}
	_, err := uc.Handle(context.Background(), apppractice.FinishSessionInput{UserID: "other", SessionID: "s-1"})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}

func TestFinishSession_ForbiddenWhenUserEmpty(t *testing.T) {
	uc := apppractice.FinishSession{}
	_, err := uc.Handle(context.Background(), apppractice.FinishSessionInput{SessionID: "s-1"})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}

func TestFinishSession_NotFound(t *testing.T) {
	uc := apppractice.FinishSession{Sessions: newFakeSessions(), Clock: clock.NewFixed(time.Now().UTC()), UoW: fakeUoW{}, Events: &fakeEvents{}}
	_, err := uc.Handle(context.Background(), apppractice.FinishSessionInput{UserID: user, SessionID: "missing"})
	require.ErrorIs(t, err, dompractice.ErrSessionNotFound)
}
