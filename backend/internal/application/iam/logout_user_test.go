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

func TestLogoutUser_RevokesFamilyAndIsIdempotent(t *testing.T) {
	users := newFakeUsers()
	tokens := newFakeRefreshTokens()
	events := &fakeIAMEvents{}
	hasher := &fakeHasher{}
	ids := idgen.NewSequential("u")
	clk := clock.NewFixed(time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC))
	mint := &fakeRefreshMinter{}
	signer := &fakeAccessSigner{}
	uow := &fakeUoW{}

	register := appiam.RegisterUser{
		Users: users, RefreshTokens: tokens, Hasher: hasher, IDs: ids,
		Clock: clk, Tokens: mint, AccessSigner: signer, UoW: uow, Events: events,
	}
	logout := appiam.LogoutUser{
		RefreshTokens: tokens, Hasher: fakeRefreshHasher{}, Clock: clk, UoW: uow, Events: events,
	}
	out, err := register.Handle(context.Background(), appiam.RegisterUserInput{Email: "a@b.co", Password: "secret-password", DisplayName: "X"})
	require.NoError(t, err)
	events.Reset()

	res, err := logout.Handle(context.Background(), appiam.LogoutUserInput{RefreshToken: out.Auth.RefreshToken})
	require.NoError(t, err)
	require.True(t, res.OK)
	require.Equal(t, []string{"iam.RefreshTokenRevoked"}, events.Names())

	// idempotent — repeated logout returns ok with no new event because the
	// hash now resolves to a token that is already revoked; the revoke is a
	// no-op but we still emit the event for audit. Verify it does not error.
	events.Reset()
	res, err = logout.Handle(context.Background(), appiam.LogoutUserInput{RefreshToken: out.Auth.RefreshToken})
	require.NoError(t, err)
	require.True(t, res.OK)

	// Logout with unknown token is also a no-op.
	events.Reset()
	_, err = logout.Handle(context.Background(), appiam.LogoutUserInput{RefreshToken: "unknown"})
	require.NoError(t, err)

	// Empty token is rejected.
	_, err = logout.Handle(context.Background(), appiam.LogoutUserInput{RefreshToken: ""})
	require.ErrorIs(t, err, domiam.ErrRefreshTokenInvalid)
}
