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

func TestGetResults_HappyPath(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	events := &fakeEvents{}
	clk := clock.NewFixed(time.Now().UTC())
	_, err := (apppractice.RateCard{Sessions: sess, Clock: clk, UoW: fakeUoW{}, Events: events}).Handle(
		context.Background(), apppractice.RateCardInput{UserID: user, SessionID: "s-1", CardID: "c-1", Rating: 2})
	require.NoError(t, err)
	_, err = (apppractice.FinishSession{Sessions: sess, Clock: clk, UoW: fakeUoW{}, Events: events}).Handle(
		context.Background(), apppractice.FinishSessionInput{UserID: user, SessionID: "s-1"})
	require.NoError(t, err)

	uc := apppractice.GetResults{Sessions: sess}
	out, err := uc.Handle(context.Background(), apppractice.GetResultsInput{UserID: user, SessionID: "s-1"})
	require.NoError(t, err)
	require.Equal(t, "s-1", out.SessionID)
	require.Equal(t, 1, out.CountKnowKnow)
	require.Len(t, out.RatedCards, 1)
}

func TestGetResults_NotCompleted(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	uc := apppractice.GetResults{Sessions: sess}
	_, err := uc.Handle(context.Background(), apppractice.GetResultsInput{UserID: user, SessionID: "s-1"})
	require.ErrorIs(t, err, dompractice.ErrSessionNotCompleted)
}

func TestGetResults_ForbiddenStranger(t *testing.T) {
	sess := newFakeSessions()
	seedSession(t, sess, dompractice.ModeTracked)
	uc := apppractice.GetResults{Sessions: sess}
	_, err := uc.Handle(context.Background(), apppractice.GetResultsInput{UserID: "other", SessionID: "s-1"})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}

func TestGetResults_NotFound(t *testing.T) {
	uc := apppractice.GetResults{Sessions: newFakeSessions()}
	_, err := uc.Handle(context.Background(), apppractice.GetResultsInput{UserID: user, SessionID: "missing"})
	require.ErrorIs(t, err, dompractice.ErrSessionNotFound)
}

func TestGetResults_ForbiddenWhenUserEmpty(t *testing.T) {
	uc := apppractice.GetResults{}
	_, err := uc.Handle(context.Background(), apppractice.GetResultsInput{SessionID: "s-1"})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}
