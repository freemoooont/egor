package iam_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/domain/iam"
)

const fakeBcrypt = "$2a$10$" + "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMN"

func mustEmail(t *testing.T, raw string) iam.EmailAddress {
	t.Helper()
	e, err := iam.NewEmailAddress(raw)
	require.NoError(t, err)
	return e
}

func mustHash(t *testing.T) iam.PasswordHash {
	t.Helper()
	h, err := iam.NewPasswordHash(fakeBcrypt)
	require.NoError(t, err)
	return h
}

func mustName(t *testing.T, raw string) iam.DisplayName {
	t.Helper()
	n, err := iam.NewDisplayName(raw)
	require.NoError(t, err)
	return n
}

// Invariant 1.
func TestUser_EmailAddressMustBeNonEmptyAfterTrim(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t\n"} {
		_, err := iam.NewEmailAddress(raw)
		require.ErrorIs(t, err, iam.ErrInvalidEmail, "raw=%q", raw)
	}
}

// Invariant 2.
func TestUser_EmailAddressMustBeRFC5321Valid(t *testing.T) {
	for _, raw := range []string{"foo", "foo@", "@bar.com", "foo bar@baz.com"} {
		_, err := iam.NewEmailAddress(raw)
		require.ErrorIs(t, err, iam.ErrInvalidEmail, "raw=%q", raw)
	}
}

// Invariant 3.
func TestUser_EmailAddressIsLowerCasedOnConstruction(t *testing.T) {
	e, err := iam.NewEmailAddress("Foo@Bar.COM")
	require.NoError(t, err)
	require.Equal(t, "foo@bar.com", e.String())
}

// Invariant 5.
func TestUser_PasswordHashMustComeFromBcryptHasher(t *testing.T) {
	for _, raw := range []string{"", "   ", "plaintext", "md5:abcdef"} {
		_, err := iam.NewPasswordHash(raw)
		require.ErrorIs(t, err, iam.ErrInvalidPasswordHash, "raw=%q", raw)
	}
	_, err := iam.NewPasswordHash(fakeBcrypt)
	require.NoError(t, err)
}

// Invariant 6.
func TestUser_DisplayNameLengthBetween1And64(t *testing.T) {
	_, err := iam.NewDisplayName("")
	require.ErrorIs(t, err, iam.ErrInvalidDisplayName)
	_, err = iam.NewDisplayName(strings.Repeat("a", 65))
	require.ErrorIs(t, err, iam.ErrInvalidDisplayName)
	d, err := iam.NewDisplayName("Vladislav")
	require.NoError(t, err)
	require.Equal(t, "Vladislav", d.String())
}

// Invariant 11.
func TestUser_AvatarRefDefaultsToNoneOnRegistration(t *testing.T) {
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	u, err := iam.NewUser("u-1", mustEmail(t, "a@b.co"), mustHash(t), mustName(t, "Eg"), now)
	require.NoError(t, err)
	require.Equal(t, iam.AvatarRefNone, u.Avatar().Kind)
	require.True(t, u.RegisteredAt().Equal(now))
}

func TestUser_SetEmailAndDisplayNameAndAvatar(t *testing.T) {
	now := time.Now().UTC()
	u, err := iam.NewUser("u-1", mustEmail(t, "a@b.co"), mustHash(t), mustName(t, "Eg"), now)
	require.NoError(t, err)

	require.NoError(t, u.SetEmail(mustEmail(t, "c@d.co")))
	require.Equal(t, "c@d.co", u.Email().String())

	require.NoError(t, u.SetDisplayName(mustName(t, "New Name")))
	require.Equal(t, "New Name", u.DisplayName().String())

	u.SetAvatar(iam.AvatarRef{Kind: iam.AvatarRefServer, Ref: "k/123"})
	require.Equal(t, iam.AvatarRefServer, u.Avatar().Kind)

	require.ErrorIs(t, u.SetEmail(iam.EmailAddress{}), iam.ErrInvalidEmail)
	require.ErrorIs(t, u.SetDisplayName(iam.DisplayName{}), iam.ErrInvalidDisplayName)
}

func TestUser_HydrateUserUsesProvidedAvatar(t *testing.T) {
	now := time.Now().UTC()
	u := iam.HydrateUser(
		"u-2",
		mustEmail(t, "x@y.co"),
		mustHash(t),
		mustName(t, "Hydra"),
		iam.AvatarRef{Kind: iam.AvatarRefServer, Ref: "kkk"},
		now,
	)
	require.Equal(t, "u-2", u.ID())
	require.Equal(t, "x@y.co", u.Email().String())
	require.Equal(t, iam.AvatarRefServer, u.Avatar().Kind)
}

func TestUser_NewUserRejectsZeroValues(t *testing.T) {
	now := time.Now().UTC()
	_, err := iam.NewUser("", mustEmail(t, "a@b.co"), mustHash(t), mustName(t, "Eg"), now)
	require.ErrorIs(t, err, iam.ErrUserNotFound)

	_, err = iam.NewUser("u-1", iam.EmailAddress{}, mustHash(t), mustName(t, "Eg"), now)
	require.ErrorIs(t, err, iam.ErrInvalidEmail)

	_, err = iam.NewUser("u-1", mustEmail(t, "a@b.co"), iam.PasswordHash{}, mustName(t, "Eg"), now)
	require.ErrorIs(t, err, iam.ErrInvalidPasswordHash)

	_, err = iam.NewUser("u-1", mustEmail(t, "a@b.co"), mustHash(t), iam.DisplayName{}, now)
	require.ErrorIs(t, err, iam.ErrInvalidDisplayName)
}

func TestUser_ChangePasswordAcceptsHashAndRejectsZero(t *testing.T) {
	now := time.Now().UTC()
	u, err := iam.NewUser("u-1", mustEmail(t, "a@b.co"), mustHash(t), mustName(t, "Eg"), now)
	require.NoError(t, err)

	other, err := iam.NewPasswordHash("$2b$10$" + strings.Repeat("a", 53))
	require.NoError(t, err)
	require.NoError(t, u.ChangePassword(other))
	require.Equal(t, other.String(), u.PasswordHash().String())

	require.ErrorIs(t, u.ChangePassword(iam.PasswordHash{}), iam.ErrInvalidPasswordHash)
}

// Invariant 8.
func TestRefreshTokenFamily_TokensAreOrderedByIssuedAt(t *testing.T) {
	base := time.Now().UTC()
	fam := iam.RefreshTokenFamily{FamilyID: "f-1", UserID: "u-1"}
	require.NoError(t, fam.AppendIssued(iam.RefreshToken{ID: "t-1", FamilyID: "f-1", UserID: "u-1", IssuedAt: base}))
	require.NoError(t, fam.AppendIssued(iam.RefreshToken{ID: "t-2", FamilyID: "f-1", UserID: "u-1", IssuedAt: base.Add(time.Second)}))

	// non-monotonic insert is rejected
	bad := fam
	require.Error(t, bad.AppendIssued(iam.RefreshToken{ID: "t-3", FamilyID: "f-1", IssuedAt: base}))
}

// Invariant 9 helper: family.Latest returns the newest, callers compare with the consumed.
func TestRefreshTokenFamily_LatestReturnsTheLastIssued(t *testing.T) {
	base := time.Now().UTC()
	fam := iam.RefreshTokenFamily{FamilyID: "f", UserID: "u"}
	_, ok := fam.Latest()
	require.False(t, ok)

	require.NoError(t, fam.AppendIssued(iam.RefreshToken{ID: "t-1", FamilyID: "f", UserID: "u", IssuedAt: base}))
	require.NoError(t, fam.AppendIssued(iam.RefreshToken{ID: "t-2", FamilyID: "f", UserID: "u", IssuedAt: base.Add(time.Second)}))
	latest, ok := fam.Latest()
	require.True(t, ok)
	require.Equal(t, "t-2", latest.ID)
}

// Invariant 10.
func TestRefreshToken_ExpiredTokenCannotMint(t *testing.T) {
	base := time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)
	tok := iam.RefreshToken{IssuedAt: base, ExpiresAt: base.Add(7 * 24 * time.Hour)}
	require.False(t, tok.IsExpired(base.Add(24*time.Hour)))
	require.True(t, tok.IsExpired(base.Add(8*24*time.Hour)))
}

func TestRefreshToken_RevokeIsIdempotent(t *testing.T) {
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	tok := iam.RefreshToken{ID: "t", IssuedAt: now, ExpiresAt: now.Add(7 * 24 * time.Hour)}
	require.False(t, tok.IsRevoked())
	tok.Revoke(now.Add(time.Minute), iam.RevokeReasonRotation)
	require.True(t, tok.IsRevoked())
	first := *tok.RevokedAt
	tok.Revoke(now.Add(time.Hour), iam.RevokeReasonReuseDetected)
	require.True(t, tok.RevokedAt.Equal(first), "second Revoke must not overwrite")
}

func TestRefreshTokenFamily_HasActiveAfterRevocation(t *testing.T) {
	base := time.Now().UTC()
	fam := iam.RefreshTokenFamily{FamilyID: "f", UserID: "u"}
	require.NoError(t, fam.AppendIssued(iam.RefreshToken{ID: "t-1", IssuedAt: base, ExpiresAt: base.Add(7 * 24 * time.Hour)}))
	require.True(t, fam.HasActiveAfterRevocation(base))

	at := base.Add(time.Minute)
	tok := &fam.Tokens[0]
	tok.Revoke(at, iam.RevokeReasonRotation)
	require.False(t, fam.HasActiveAfterRevocation(at))
}
