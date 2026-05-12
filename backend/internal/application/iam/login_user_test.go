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

func newLoginEnv(t *testing.T) (appiam.RegisterUser, appiam.LoginUser, *fakeIAMEvents) {
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
	login := appiam.LoginUser{
		Users: users, RefreshTokens: tokens, Hasher: hasher, IDs: ids,
		Clock: clk, Tokens: mint, AccessSigner: signer, UoW: uow, Events: events,
	}
	return register, login, events
}

func TestLoginUser_HappyPath(t *testing.T) {
	register, login, events := newLoginEnv(t)
	_, err := register.Handle(context.Background(), appiam.RegisterUserInput{Email: "a@b.co", Password: "secret-password", DisplayName: "X"})
	require.NoError(t, err)
	events.Reset()

	out, err := login.Handle(context.Background(), appiam.LoginUserInput{Email: "A@B.co", Password: "secret-password"})
	require.NoError(t, err)
	require.Equal(t, "a@b.co", out.User.Email)
	require.NotEmpty(t, out.Auth.AccessToken)
	require.Equal(t, []string{"iam.UserLoggedIn", "iam.RefreshTokenIssued"}, events.Names())
}

func TestLoginUser_InvalidCredentialsForUnknownUser(t *testing.T) {
	_, login, _ := newLoginEnv(t)
	_, err := login.Handle(context.Background(), appiam.LoginUserInput{Email: "ghost@b.co", Password: "secret-password"})
	require.ErrorIs(t, err, domiam.ErrInvalidCredentials)
}

func TestLoginUser_InvalidCredentialsForBadPassword(t *testing.T) {
	register, login, _ := newLoginEnv(t)
	_, err := register.Handle(context.Background(), appiam.RegisterUserInput{Email: "a@b.co", Password: "secret-password", DisplayName: "X"})
	require.NoError(t, err)
	_, err = login.Handle(context.Background(), appiam.LoginUserInput{Email: "a@b.co", Password: "wrong-password"})
	require.ErrorIs(t, err, domiam.ErrInvalidCredentials)
}

func TestLoginUser_InvalidEmailFormat(t *testing.T) {
	_, login, _ := newLoginEnv(t)
	_, err := login.Handle(context.Background(), appiam.LoginUserInput{Email: "nope", Password: "secret-password"})
	require.ErrorIs(t, err, domiam.ErrInvalidEmail)
}
