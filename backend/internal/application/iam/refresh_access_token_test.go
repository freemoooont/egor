package iam_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appiam "github.com/micocards/api/internal/application/iam"
	domiam "github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/idgen"
)

type rotEnv struct {
	register appiam.RegisterUser
	rotate   appiam.RefreshAccessToken
	tokens   *fakeRefreshTokens
	events   *fakeIAMEvents
	clk      *clock.Fixed
}

func newRotateEnv(t *testing.T) rotEnv {
	t.Helper()
	users := newFakeUsers()
	tokens := newFakeRefreshTokens()
	hasher := &fakeHasher{}
	ids := idgen.NewSequential("u")
	clk := clock.NewFixed(time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC))
	mint := &fakeRefreshMinter{}
	signer := &fakeAccessSigner{}
	uow := &fakeUoW{}
	events := &fakeIAMEvents{}

	register := appiam.RegisterUser{
		Users: users, RefreshTokens: tokens, Hasher: hasher, IDs: ids,
		Clock: clk, Tokens: mint, AccessSigner: signer, UoW: uow, Events: events,
	}
	rotate := appiam.RefreshAccessToken{
		RefreshTokens: tokens, IDs: ids, Clock: clk, Tokens: mint,
		AccessSigner: signer, UoW: uow, Events: events,
		Hasher: fakeRefreshHasher{},
	}
	return rotEnv{register: register, rotate: rotate, tokens: tokens, events: events, clk: clk}
}

func TestRefreshAccessToken_RotatesAndRevokesPrevious(t *testing.T) {
	env := newRotateEnv(t)
	regOut, err := env.register.Handle(context.Background(), appiam.RegisterUserInput{Email: "a@b.co", Password: "secret-password", DisplayName: "X"})
	require.NoError(t, err)
	env.events.Reset()

	rotOut, err := env.rotate.Handle(context.Background(), appiam.RefreshAccessTokenInput{RefreshToken: regOut.Auth.RefreshToken})
	require.NoError(t, err)
	require.NotEmpty(t, rotOut.Auth.RefreshToken)
	require.NotEqual(t, regOut.Auth.RefreshToken, rotOut.Auth.RefreshToken)

	require.Equal(t, []string{"iam.RefreshTokenRevoked", "iam.RefreshTokenIssued"}, env.events.Names())

	// previous token now revoked
	prevHash := sha256Hex(regOut.Auth.RefreshToken)
	prev, err := env.tokens.ByOpaqueHash(context.Background(), prevHash)
	require.NoError(t, err)
	require.True(t, prev.IsRevoked())
}

func TestRefreshAccessToken_ReuseDetectionRevokesFamily(t *testing.T) {
	env := newRotateEnv(t)
	regOut, err := env.register.Handle(context.Background(), appiam.RegisterUserInput{Email: "a@b.co", Password: "secret-password", DisplayName: "X"})
	require.NoError(t, err)
	env.events.Reset()

	// first rotation succeeds
	_, err = env.rotate.Handle(context.Background(), appiam.RefreshAccessTokenInput{RefreshToken: regOut.Auth.RefreshToken})
	require.NoError(t, err)
	env.events.Reset()

	// replay of the original refresh = reuse detection
	_, err = env.rotate.Handle(context.Background(), appiam.RefreshAccessTokenInput{RefreshToken: regOut.Auth.RefreshToken})
	require.ErrorIs(t, err, domiam.ErrRefreshTokenReused)

	// every token in the family should now be revoked
	famID := ""
	for _, tok := range env.tokens.byID {
		famID = tok.FamilyID
		break
	}
	fam, err := env.tokens.FamilyByID(context.Background(), famID)
	require.NoError(t, err)
	for _, t2 := range fam.Tokens {
		require.True(t, t2.IsRevoked(), "token %s should be revoked after reuse detection", t2.ID)
	}
}

func TestRefreshAccessToken_ExpiredReturnsErrRefreshTokenExpired(t *testing.T) {
	env := newRotateEnv(t)
	regOut, err := env.register.Handle(context.Background(), appiam.RegisterUserInput{Email: "a@b.co", Password: "secret-password", DisplayName: "X"})
	require.NoError(t, err)

	env.clk.Advance(8 * 24 * time.Hour)
	_, err = env.rotate.Handle(context.Background(), appiam.RefreshAccessTokenInput{RefreshToken: regOut.Auth.RefreshToken})
	require.ErrorIs(t, err, domiam.ErrRefreshTokenExpired)
}

func TestRefreshAccessToken_UnknownTokenReturnsInvalid(t *testing.T) {
	env := newRotateEnv(t)
	_, err := env.rotate.Handle(context.Background(), appiam.RefreshAccessTokenInput{RefreshToken: "unknown"})
	require.ErrorIs(t, err, domiam.ErrRefreshTokenInvalid)
}

func TestRefreshAccessToken_EmptyTokenInvalid(t *testing.T) {
	env := newRotateEnv(t)
	_, err := env.rotate.Handle(context.Background(), appiam.RefreshAccessTokenInput{RefreshToken: ""})
	require.ErrorIs(t, err, domiam.ErrRefreshTokenInvalid)
}
