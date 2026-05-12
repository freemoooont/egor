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

func newRegisterUC(t *testing.T) (appiam.RegisterUser, *fakeUsers, *fakeRefreshTokens, *fakeIAMEvents) {
	t.Helper()
	users := newFakeUsers()
	tokens := newFakeRefreshTokens()
	events := &fakeIAMEvents{}
	uc := appiam.RegisterUser{
		Users:         users,
		RefreshTokens: tokens,
		Hasher:        &fakeHasher{},
		IDs:           idgen.NewSequential("u"),
		Clock:         clock.NewFixed(time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)),
		Tokens:        &fakeRefreshMinter{},
		AccessSigner:  &fakeAccessSigner{},
		UoW:           &fakeUoW{},
		Events:        events,
	}
	return uc, users, tokens, events
}

func TestRegisterUser_HappyPath(t *testing.T) {
	uc, users, tokens, events := newRegisterUC(t)
	out, err := uc.Handle(context.Background(), appiam.RegisterUserInput{
		Email:       "Foo@Bar.com",
		Password:    "secret-password-123",
		DisplayName: "Foo",
	})
	require.NoError(t, err)
	require.Equal(t, "foo@bar.com", out.User.Email)
	require.Equal(t, "Foo", out.User.DisplayName)
	require.Equal(t, domiam.AvatarRefNone, out.User.AvatarKind)
	require.NotEmpty(t, out.Auth.AccessToken)
	require.NotEmpty(t, out.Auth.RefreshToken)
	require.True(t, out.Auth.RefreshTokenExpiresAt.After(out.Auth.AccessTokenExpiresAt))

	require.Len(t, users.byID, 1)
	require.Len(t, tokens.byID, 1)
	require.Equal(t, []string{"iam.UserRegistered", "iam.RefreshTokenIssued"}, events.Names())
}

func TestRegisterUser_DuplicateEmailReturnsErrEmailTaken(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	in := appiam.RegisterUserInput{Email: "a@b.co", Password: "secret-password-123", DisplayName: "X"}
	_, err := uc.Handle(context.Background(), in)
	require.NoError(t, err)
	_, err = uc.Handle(context.Background(), in)
	require.ErrorIs(t, err, domiam.ErrEmailTaken)
}

func TestRegisterUser_RejectsInvalidEmail(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Handle(context.Background(), appiam.RegisterUserInput{Email: "nope", Password: "secret-password", DisplayName: "X"})
	require.ErrorIs(t, err, domiam.ErrInvalidEmail)
}

func TestRegisterUser_RejectsWeakPassword(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Handle(context.Background(), appiam.RegisterUserInput{Email: "a@b.co", Password: "short", DisplayName: "X"})
	require.ErrorIs(t, err, domiam.ErrPasswordTooWeak)
}

func TestRegisterUser_RejectsEmptyDisplayName(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)
	_, err := uc.Handle(context.Background(), appiam.RegisterUserInput{Email: "a@b.co", Password: "secret-password", DisplayName: " "})
	require.ErrorIs(t, err, domiam.ErrInvalidDisplayName)
}
