package practice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppractice "github.com/micocards/api/internal/application/practice"
	dompractice "github.com/micocards/api/internal/domain/practice"
)

func TestGetUserDeckProgress_HappyPath(t *testing.T) {
	prog := newFakeProgress()
	now := time.Now().UTC()
	require.NoError(t, prog.Save(context.Background(), &dompractice.UserDeckProgress{
		UserID: user, DeckID: deck,
		Cards: []dompractice.CardProgress{
			{CardID: "c-1", Rating: dompractice.RatingKnowKnow, LastRatedAt: now},
		},
		UpdatedAt: now,
	}))
	uc := apppractice.GetUserDeckProgress{Progress: prog}
	out, err := uc.Handle(context.Background(), apppractice.GetUserDeckProgressInput{UserID: user, DeckID: deck})
	require.NoError(t, err)
	require.Equal(t, deck, out.DeckID)
	require.Len(t, out.CardProgress, 1)
	require.Equal(t, int16(2), out.CardProgress[0].Rating)
}

func TestGetUserDeckProgress_EmptyWhenMissing(t *testing.T) {
	uc := apppractice.GetUserDeckProgress{Progress: newFakeProgress()}
	out, err := uc.Handle(context.Background(), apppractice.GetUserDeckProgressInput{UserID: user, DeckID: deck})
	require.NoError(t, err)
	require.Equal(t, deck, out.DeckID)
	require.Empty(t, out.CardProgress)
}

func TestGetUserDeckProgress_ForbiddenWhenUserEmpty(t *testing.T) {
	uc := apppractice.GetUserDeckProgress{Progress: newFakeProgress()}
	_, err := uc.Handle(context.Background(), apppractice.GetUserDeckProgressInput{DeckID: deck})
	require.ErrorIs(t, err, dompractice.ErrForbidden)
}
