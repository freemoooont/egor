package decks

import (
	"context"

	"github.com/micocards/api/internal/domain/decks"
	"github.com/micocards/api/internal/domain/shared"
)

// GenerateDeckWithAI is the AI-generation stub. v1 returns 501 / not_implemented
// when AIProvider.IsConfigured() is false.
type GenerateDeckWithAI struct {
	AI     AIProvider
	IDs    shared.IDGenerator
	Clock  shared.Clock
	Events EventPublisher
}

// Handle calls the provider when configured; otherwise emits a "not_implemented"
// completion event and surfaces ErrAINotConfigured to the caller.
func (uc GenerateDeckWithAI) Handle(ctx context.Context, in GenerateDeckWithAIInput) (GenerateDeckWithAIOutput, error) {
	now := uc.Clock.Now(ctx)
	requestID := in.RequestID
	if requestID == "" {
		requestID = uc.IDs.NewID(ctx)
	}
	requested := decks.DeckGenerationRequested{
		RequestID: requestID, OwnerID: in.OwnerID, Prompt: in.Prompt, RequestedAt: now,
	}
	if err := uc.Events.Publish(ctx, requested); err != nil {
		return GenerateDeckWithAIOutput{}, err
	}

	if !uc.AI.IsConfigured() {
		_ = uc.Events.Publish(ctx, decks.DeckGenerationCompleted{
			RequestID: requestID, OwnerID: in.OwnerID,
			Status: "not_implemented", CompletedAt: now,
		})
		return GenerateDeckWithAIOutput{}, decks.ErrAINotConfigured
	}

	draft, err := uc.AI.Generate(ctx, in.Prompt)
	if err != nil {
		_ = uc.Events.Publish(ctx, decks.DeckGenerationCompleted{
			RequestID: requestID, OwnerID: in.OwnerID,
			Status: "failed", CompletedAt: now,
		})
		return GenerateDeckWithAIOutput{}, decks.ErrAIUpstream
	}

	view := AIDeckDraftView{Title: draft.Title}
	for _, c := range draft.Cards {
		view.Cards = append(view.Cards, DraftCard{Term: c.Term, Definition: c.Definition})
	}
	if err := uc.Events.Publish(ctx, decks.DeckGenerationCompleted{
		RequestID: requestID, OwnerID: in.OwnerID,
		Status: "ok", Draft: draft, CompletedAt: now,
	}); err != nil {
		return GenerateDeckWithAIOutput{}, err
	}
	return GenerateDeckWithAIOutput{Status: "ok", Draft: view}, nil
}
