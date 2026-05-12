package clock_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/infrastructure/clock"
)

func TestSystem_NowIsRecentAndUTC(t *testing.T) {
	got := clock.System{}.Now(context.Background())
	require.WithinDuration(t, time.Now().UTC(), got, time.Second)
	require.Equal(t, time.UTC, got.Location())
}

func TestFixed_AdvanceAndSet(t *testing.T) {
	at := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	f := clock.NewFixed(at)
	require.Equal(t, at, f.Now(context.Background()))
	f.Advance(time.Hour)
	require.Equal(t, at.Add(time.Hour), f.Now(context.Background()))
	other := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	f.Set(other)
	require.Equal(t, other, f.Now(context.Background()))
}
