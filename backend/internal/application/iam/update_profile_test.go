package iam_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appiam "github.com/micocards/api/internal/application/iam"
	domiam "github.com/micocards/api/internal/domain/iam"
)

func seedUser(t *testing.T, users *fakeUsers, id, email, name string) {
	t.Helper()
	hasher := &fakeHasher{}
	ph, err := hasher.Hash(context.Background(), "secret-password")
	require.NoError(t, err)
	em, err := domiam.NewEmailAddress(email)
	require.NoError(t, err)
	nm, err := domiam.NewDisplayName(name)
	require.NoError(t, err)
	u, err := domiam.NewUser(id, em, ph, nm, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, users.Save(context.Background(), u))
}

func TestUpdateProfile_HappyPath(t *testing.T) {
	users := newFakeUsers()
	seedUser(t, users, "u-1", "a@b.co", "Old")
	uc := appiam.UpdateProfile{Users: users, UoW: &fakeUoW{}}
	newName := "New Name"
	newEmail := "x@y.co"
	out, err := uc.Handle(context.Background(), appiam.UpdateProfileInput{
		UserID:      "u-1",
		DisplayName: &newName,
		Email:       &newEmail,
	})
	require.NoError(t, err)
	require.Equal(t, "New Name", out.User.DisplayName)
	require.Equal(t, "x@y.co", out.User.Email)
}

func TestUpdateProfile_UnauthorizedWhenIDEmpty(t *testing.T) {
	uc := appiam.UpdateProfile{Users: newFakeUsers(), UoW: &fakeUoW{}}
	_, err := uc.Handle(context.Background(), appiam.UpdateProfileInput{})
	require.ErrorIs(t, err, domiam.ErrUnauthorized)
}

func TestUpdateProfile_RejectsInvalidEmail(t *testing.T) {
	users := newFakeUsers()
	seedUser(t, users, "u-1", "a@b.co", "Old")
	uc := appiam.UpdateProfile{Users: users, UoW: &fakeUoW{}}
	bad := "nope"
	_, err := uc.Handle(context.Background(), appiam.UpdateProfileInput{UserID: "u-1", Email: &bad})
	require.ErrorIs(t, err, domiam.ErrInvalidEmail)
}

func TestUpdateProfile_RejectsTakenEmail(t *testing.T) {
	users := newFakeUsers()
	seedUser(t, users, "u-1", "a@b.co", "Old")
	seedUser(t, users, "u-2", "x@y.co", "Other")
	uc := appiam.UpdateProfile{Users: users, UoW: &fakeUoW{}}
	taken := "x@y.co"
	_, err := uc.Handle(context.Background(), appiam.UpdateProfileInput{UserID: "u-1", Email: &taken})
	require.ErrorIs(t, err, domiam.ErrEmailTaken)
}

func TestUpdateProfile_RejectsInvalidName(t *testing.T) {
	users := newFakeUsers()
	seedUser(t, users, "u-1", "a@b.co", "Old")
	uc := appiam.UpdateProfile{Users: users, UoW: &fakeUoW{}}
	bad := ""
	_, err := uc.Handle(context.Background(), appiam.UpdateProfileInput{UserID: "u-1", DisplayName: &bad})
	require.ErrorIs(t, err, domiam.ErrInvalidDisplayName)
}

func TestUpdateProfile_NoOpWhenSameEmail(t *testing.T) {
	users := newFakeUsers()
	seedUser(t, users, "u-1", "a@b.co", "Old")
	uc := appiam.UpdateProfile{Users: users, UoW: &fakeUoW{}}
	same := "a@b.co"
	out, err := uc.Handle(context.Background(), appiam.UpdateProfileInput{UserID: "u-1", Email: &same})
	require.NoError(t, err)
	require.Equal(t, "a@b.co", out.User.Email)
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	uc := appiam.UpdateProfile{Users: newFakeUsers(), UoW: &fakeUoW{}}
	_, err := uc.Handle(context.Background(), appiam.UpdateProfileInput{UserID: "missing"})
	require.ErrorIs(t, err, domiam.ErrUserNotFound)
}
