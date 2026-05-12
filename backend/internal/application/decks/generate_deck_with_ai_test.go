package decks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appdecks "github.com/micocards/api/internal/application/decks"
	domdecks "github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
)

func TestGenerateDeckWithAI_NotConfiguredReturnsSentinel(t *testing.T) {
	events := &fakeDeckEvents{}
	uc := appdecks.GenerateDeckWithAI{
		AI:     fakeAI{configured: false},
		IDs:    idgen.NewSequential("r"),
		Clock:  clock.NewFixed(time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)),
		Events: events,
	}
	_, err := uc.Handle(context.Background(), appdecks.GenerateDeckWithAIInput{OwnerID: owner, Prompt: "p"})
	require.ErrorIs(t, err, domdecks.ErrAINotConfigured)
	require.Equal(t, []string{"decks.DeckGenerationRequested", "decks.DeckGenerationCompleted"}, events.Names())
}

func TestGenerateDeckWithAI_ConfiguredHappyPath(t *testing.T) {
	events := &fakeDeckEvents{}
	uc := appdecks.GenerateDeckWithAI{
		AI: fakeAI{configured: true, draft: domdecks.AIDeckDraft{
			Title: "AI Title",
			Cards: []domdecks.AIDraftCard{{Term: "t", Definition: "d"}},
		}},
		IDs:    idgen.NewSequential("r"),
		Clock:  clock.NewFixed(time.Now().UTC()),
		Events: events,
	}
	out, err := uc.Handle(context.Background(), appdecks.GenerateDeckWithAIInput{OwnerID: owner, Prompt: "p", RequestID: "r-1"})
	require.NoError(t, err)
	require.Equal(t, "ok", out.Status)
	require.Equal(t, "AI Title", out.Draft.Title)
	require.Len(t, out.Draft.Cards, 1)
}

func TestGenerateDeckWithAI_UpstreamError(t *testing.T) {
	events := &fakeDeckEvents{}
	uc := appdecks.GenerateDeckWithAI{
		AI:     fakeAI{configured: true, err: errors.New("upstream")},
		IDs:    idgen.NewSequential("r"),
		Clock:  clock.NewFixed(time.Now().UTC()),
		Events: events,
	}
	_, err := uc.Handle(context.Background(), appdecks.GenerateDeckWithAIInput{OwnerID: owner, Prompt: "p"})
	require.ErrorIs(t, err, domdecks.ErrAIUpstream)
	require.Contains(t, events.Names(), "decks.DeckGenerationCompleted")
}
