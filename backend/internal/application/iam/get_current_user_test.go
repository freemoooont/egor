package iam_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appiam "github.com/micocards/api/internal/application/iam"
	domiam "github.com/micocards/api/internal/domain/iam"
)

func TestGetCurrentUser_Success(t *testing.T) {
	users := newFakeUsers()
	hasher := &fakeHasher{}
	ph, err := hasher.Hash(context.Background(), "secret-password")
	require.NoError(t, err)
	email, err := domiam.NewEmailAddress("a@b.co")
	require.NoError(t, err)
	name, err := domiam.NewDisplayName("Egor")
	require.NoError(t, err)
	u, err := domiam.NewUser("u-1", email, ph, name, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, users.Save(context.Background(), u))

	uc := appiam.GetCurrentUser{Users: users}
	out, err := uc.Handle(context.Background(), appiam.GetCurrentUserInput{UserID: "u-1"})
	require.NoError(t, err)
	require.Equal(t, "a@b.co", out.User.Email)
	require.Equal(t, "Egor", out.User.DisplayName)
}

func TestGetCurrentUser_UnauthorizedWhenIDEmpty(t *testing.T) {
	uc := appiam.GetCurrentUser{Users: newFakeUsers()}
	_, err := uc.Handle(context.Background(), appiam.GetCurrentUserInput{UserID: ""})
	require.ErrorIs(t, err, domiam.ErrUnauthorized)
}

func TestGetCurrentUser_UserNotFound(t *testing.T) {
	uc := appiam.GetCurrentUser{Users: newFakeUsers()}
	_, err := uc.Handle(context.Background(), appiam.GetCurrentUserInput{UserID: "missing"})
	require.ErrorIs(t, err, domiam.ErrUserNotFound)
}
