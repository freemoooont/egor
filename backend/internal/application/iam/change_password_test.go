package iam_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appiam "github.com/micocards/api/internal/application/iam"
	domiam "github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/infrastructure/clock"
)

// Invariant 7: User_ChangePasswordRotatesHashAndRevokesAllRefreshFamilies.
func TestChangePassword_RotatesHashAndRevokesRefreshFamilies(t *testing.T) {
	users := newFakeUsers()
	tokens := newFakeRefreshTokens()
	hasher := &fakeHasher{}
	ph, err := hasher.Hash(context.Background(), "old-password-123")
	require.NoError(t, err)
	em, err := domiam.NewEmailAddress("a@b.co")
	require.NoError(t, err)
	nm, err := domiam.NewDisplayName("X")
	require.NoError(t, err)
	u, err := domiam.NewUser("u-1", em, ph, nm, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, users.Save(context.Background(), u))

	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	require.NoError(t, tokens.Save(context.Background(), domiam.RefreshToken{
		ID: "t-1", FamilyID: "f-1", UserID: "u-1", OpaqueHash: "h-1",
		IssuedAt: now, ExpiresAt: now.Add(7 * 24 * time.Hour),
	}))

	events := &fakeIAMEvents{}
	uc := appiam.ChangePassword{
		Users: users, RefreshTokens: tokens, Hasher: hasher,
		Clock: clock.NewFixed(now), UoW: &fakeUoW{}, Events: events,
	}

	out, err := uc.Handle(context.Background(), appiam.ChangePasswordInput{
		UserID: "u-1", CurrentPassword: "old-password-123", NewPassword: "new-password-456",
	})
	require.NoError(t, err)
	require.True(t, out.OK)

	saved, err := users.ByID(context.Background(), "u-1")
	require.NoError(t, err)
	require.NotEqual(t, ph.String(), saved.PasswordHash().String())

	tok, err := tokens.ByOpaqueHash(context.Background(), "h-1")
	require.NoError(t, err)
	require.True(t, tok.IsRevoked())
	require.Equal(t, []string{"iam.RefreshTokenRevoked"}, events.Names())
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	users := newFakeUsers()
	hasher := &fakeHasher{}
	ph, _ := hasher.Hash(context.Background(), "old-password-123")
	em, _ := domiam.NewEmailAddress("a@b.co")
	nm, _ := domiam.NewDisplayName("X")
	u, _ := domiam.NewUser("u-1", em, ph, nm, time.Now().UTC())
	_ = users.Save(context.Background(), u)

	uc := appiam.ChangePassword{
		Users: users, RefreshTokens: newFakeRefreshTokens(), Hasher: hasher,
		Clock: clock.NewFixed(time.Now().UTC()), UoW: &fakeUoW{}, Events: &fakeIAMEvents{},
	}
	_, err := uc.Handle(context.Background(), appiam.ChangePasswordInput{
		UserID: "u-1", CurrentPassword: "wrong-password", NewPassword: "new-password-456",
	})
	require.ErrorIs(t, err, domiam.ErrInvalidCredentials)
}

func TestChangePassword_RejectsWeakNewPassword(t *testing.T) {
	uc := appiam.ChangePassword{
		Users: newFakeUsers(), RefreshTokens: newFakeRefreshTokens(), Hasher: &fakeHasher{},
		Clock: clock.NewFixed(time.Now().UTC()), UoW: &fakeUoW{}, Events: &fakeIAMEvents{},
	}
	_, err := uc.Handle(context.Background(), appiam.ChangePasswordInput{UserID: "u-1", CurrentPassword: "x", NewPassword: "short"})
	require.ErrorIs(t, err, domiam.ErrPasswordTooWeak)
}

func TestChangePassword_UnauthorizedWhenIDEmpty(t *testing.T) {
	uc := appiam.ChangePassword{Hasher: &fakeHasher{}}
	_, err := uc.Handle(context.Background(), appiam.ChangePasswordInput{NewPassword: "secret-password-123"})
	require.ErrorIs(t, err, domiam.ErrUnauthorized)
}
