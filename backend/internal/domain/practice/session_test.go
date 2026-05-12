package practice_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/domain/practice"
)

const (
	user    = "u-1"
	deck    = "d-1"
	other   = "u-2"
	session = "s-1"
)

func newTracked(t *testing.T) *practice.Session {
	t.Helper()
	s, err := practice.NewSession(session, user, deck, practice.ModeTracked, []string{"c-1", "c-2"}, time.Now().UTC())
	require.NoError(t, err)
	return s
}

// Invariant 1.
func TestSession_StatusStartsAsInProgress(t *testing.T) {
	s := newTracked(t)
	require.Equal(t, practice.StatusInProgress, s.Status())
}

// Invariant 2.
func TestSession_RateRequiresInProgressStatus(t *testing.T) {
	s := newTracked(t)
	_, err := s.Finish(time.Now().UTC())
	require.NoError(t, err)
	_, err = s.Rate("c-1", practice.RatingKnowKnow, time.Now().UTC())
	require.ErrorIs(t, err, practice.ErrSessionClosed)
}

// Invariant 3.
func TestSession_RateRequiresTrackedMode(t *testing.T) {
	s, err := practice.NewSession(session, user, deck, practice.ModeUntracked, []string{"c-1"}, time.Now().UTC())
	require.NoError(t, err)
	_, err = s.Rate("c-1", practice.RatingKnowKnow, time.Now().UTC())
	require.ErrorIs(t, err, practice.ErrSessionUntracked)
}

// Invariant 4.
func TestSession_RateRejectsCardsNotInSnapshot(t *testing.T) {
	s := newTracked(t)
	_, err := s.Rate("c-XYZ", practice.RatingDontKnow, time.Now().UTC())
	require.ErrorIs(t, err, practice.ErrCardNotInSession)
}

// Invariant 5.
func TestSession_RateIsIdempotentPerCardID(t *testing.T) {
	s := newTracked(t)
	now := time.Now().UTC()
	_, err := s.Rate("c-1", practice.RatingDontKnow, now)
	require.NoError(t, err)
	_, err = s.Rate("c-1", practice.RatingKnowKnow, now.Add(time.Second))
	require.NoError(t, err)
	rated := s.RatedCards()
	require.Len(t, rated, 1)
	require.Equal(t, practice.RatingKnowKnow, rated[0].Rating)
}

// Invariant 6.
func TestSession_FinishIsTerminalAndIdempotent(t *testing.T) {
	s := newTracked(t)
	now := time.Now().UTC()
	sum1, err := s.Finish(now)
	require.NoError(t, err)
	require.Equal(t, practice.StatusCompleted, s.Status())
	require.NotNil(t, s.CompletedAt())
	first := *s.CompletedAt()
	sum2, err := s.Finish(now.Add(time.Hour))
	require.NoError(t, err)
	require.True(t, s.CompletedAt().Equal(first))
	require.Equal(t, sum1, sum2)
}

// Invariant 7.
func TestSession_AbandonIsTerminalAndIdempotent(t *testing.T) {
	s := newTracked(t)
	now := time.Now().UTC()
	require.NoError(t, s.Abandon(now))
	require.Equal(t, practice.StatusAbandoned, s.Status())
	require.NotNil(t, s.AbandonedAt())
	first := *s.AbandonedAt()
	require.NoError(t, s.Abandon(now.Add(time.Hour)))
	require.True(t, s.AbandonedAt().Equal(first))
}

func TestSession_FinishAfterAbandonRejected(t *testing.T) {
	s := newTracked(t)
	now := time.Now().UTC()
	require.NoError(t, s.Abandon(now))
	_, err := s.Finish(now.Add(time.Minute))
	require.ErrorIs(t, err, practice.ErrSessionClosed)
}

func TestSession_AbandonAfterFinishRejected(t *testing.T) {
	s := newTracked(t)
	_, err := s.Finish(time.Now().UTC())
	require.NoError(t, err)
	require.ErrorIs(t, s.Abandon(time.Now().UTC()), practice.ErrSessionClosed)
}

// Invariant 8 + 9 (immutability after construction).
func TestSession_OwnedBySingleUser_DeckIDIsImmutable(t *testing.T) {
	s := newTracked(t)
	require.Equal(t, user, s.UserID())
	require.Equal(t, deck, s.DeckID())
	require.ErrorIs(t, s.Authorize(other), practice.ErrForbidden)
	require.NoError(t, s.Authorize(user))
}

// Invariant 10.
func TestSession_TrackedFinishEmitsCardRatedSummary(t *testing.T) {
	s, err := practice.NewSession(session, user, deck, practice.ModeTracked, []string{"a", "b", "c", "d"}, time.Now().UTC())
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = s.Rate("a", practice.RatingDontKnow, now)
	require.NoError(t, err)
	_, err = s.Rate("b", practice.RatingStillLearning, now)
	require.NoError(t, err)
	_, err = s.Rate("c", practice.RatingKnowKnow, now)
	require.NoError(t, err)
	_, err = s.Rate("d", practice.RatingKnowKnow, now)
	require.NoError(t, err)

	sum, err := s.Finish(now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 1, sum.CountDontKnow)
	require.Equal(t, 1, sum.CountStillLearning)
	require.Equal(t, 2, sum.CountKnowKnow)
	require.Len(t, s.RatedCards(), 4)
}

func TestSession_NewSessionRejectsBadInput(t *testing.T) {
	now := time.Now().UTC()
	_, err := practice.NewSession("", user, deck, practice.ModeTracked, []string{"c"}, now)
	require.ErrorIs(t, err, practice.ErrSessionNotFound)

	_, err = practice.NewSession(session, user, deck, "weird", []string{"c"}, now)
	require.ErrorIs(t, err, practice.ErrInvalidPracticeMode)

	_, err = practice.NewSession(session, user, deck, practice.ModeTracked, nil, now)
	require.ErrorIs(t, err, practice.ErrDeckEmpty)
}

func TestSession_RateInvalidRating(t *testing.T) {
	s := newTracked(t)
	_, err := s.Rate("c-1", practice.CardRating(99), time.Now().UTC())
	require.ErrorIs(t, err, practice.ErrInvalidRating)
}

func TestSession_HydrateRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(time.Hour)
	s := practice.HydrateSession(
		session, user, deck,
		practice.ModeTracked, practice.StatusCompleted,
		[]string{"c-1"},
		[]practice.RatedCard{{CardID: "c-1", Rating: practice.RatingKnowKnow, RatedAt: now}},
		now, &completed, nil,
	)
	require.Equal(t, practice.StatusCompleted, s.Status())
	require.Equal(t, completed.UTC(), *s.CompletedAt())
	require.Len(t, s.RatedCards(), 1)
	require.Equal(t, []string{"c-1"}, s.CardIDs())
	require.Equal(t, 1, s.Summary().CountKnowKnow)
}

// UserDeckProgress invariants 1, 2, 3, 4.

func TestUserDeckProgress_OneRowPerUserDeckPair_LatestRatingWins_UntrackedIgnored(t *testing.T) {
	now := time.Now().UTC()
	p := &practice.UserDeckProgress{UserID: user, DeckID: deck}
	rated1 := []practice.RatedCard{{CardID: "c-1", Rating: practice.RatingDontKnow, RatedAt: now}}
	p.ApplySessionCompleted(practice.ModeTracked, rated1, now)
	require.Len(t, p.Cards, 1)

	rated2 := []practice.RatedCard{{CardID: "c-1", Rating: practice.RatingKnowKnow, RatedAt: now.Add(time.Hour)}}
	p.ApplySessionCompleted(practice.ModeTracked, rated2, now.Add(time.Hour))
	require.Len(t, p.Cards, 1) // invariant 1 — single row
	require.Equal(t, practice.RatingKnowKnow, p.Cards[0].Rating)

	// untracked is ignored (invariant 3)
	p.ApplySessionCompleted(practice.ModeUntracked,
		[]practice.RatedCard{{CardID: "c-1", Rating: practice.RatingDontKnow, RatedAt: now.Add(2 * time.Hour)}},
		now.Add(2*time.Hour))
	require.Equal(t, practice.RatingKnowKnow, p.Cards[0].Rating)
}

func TestUserDeckProgress_DropsCardOnCardRemoved(t *testing.T) {
	now := time.Now().UTC()
	p := &practice.UserDeckProgress{
		UserID: user,
		DeckID: deck,
		Cards: []practice.CardProgress{
			{CardID: "c-1", Rating: practice.RatingDontKnow, LastRatedAt: now},
			{CardID: "c-2", Rating: practice.RatingKnowKnow, LastRatedAt: now},
		},
	}
	p.DropCard("c-1")
	require.Len(t, p.Cards, 1)
	require.Equal(t, "c-2", p.Cards[0].CardID)
}

func TestEvents_NamesArePackagePrefixed(t *testing.T) {
	cases := []practice.Event{
		practice.PracticeSessionStarted{SessionID: "s"},
		practice.CardRated{SessionID: "s", CardID: "c"},
		practice.PracticeSessionCompleted{SessionID: "s"},
	}
	want := []string{
		"practice.PracticeSessionStarted",
		"practice.CardRated",
		"practice.PracticeSessionCompleted",
	}
	for i, ev := range cases {
		require.Equal(t, want[i], ev.Name())
	}
}

func TestSessionMode_IsValid(t *testing.T) {
	require.True(t, practice.ModeTracked.IsValid())
	require.True(t, practice.ModeUntracked.IsValid())
	require.False(t, practice.SessionMode("nope").IsValid())
}

func TestCardRating_IsValid(t *testing.T) {
	require.True(t, practice.RatingDontKnow.IsValid())
	require.True(t, practice.RatingStillLearning.IsValid())
	require.True(t, practice.RatingKnowKnow.IsValid())
	require.False(t, practice.CardRating(7).IsValid())
}
