package iam

import (
	"strings"
	"time"
	"unicode/utf8"
)

// AvatarRefKind enumerates the v1 avatar storage modes.
type AvatarRefKind string

const (
	// AvatarRefNone is the default for fresh users.
	AvatarRefNone AvatarRefKind = "none"
	// AvatarRefServer points to a server-stored asset (object store key).
	AvatarRefServer AvatarRefKind = "server"
)

// AvatarRef is a tiny VO so the OpenAPI schema can stay tight.
type AvatarRef struct {
	Kind AvatarRefKind
	Ref  string // empty for none
}

// NoneAvatar is the canonical zero-value avatar.
var NoneAvatar = AvatarRef{Kind: AvatarRefNone}

// DisplayName is a length-checked Unicode-trimmed name.
type DisplayName struct{ value string }

// NewDisplayName validates the 1..64 grapheme range (we approximate "graphemes"
// with rune count, which is enough for v1).
//
// Invariant: User_DisplayNameLengthBetween1And64.
func NewDisplayName(raw string) (DisplayName, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DisplayName{}, ErrInvalidDisplayName
	}
	n := utf8.RuneCountInString(trimmed)
	if n < 1 || n > 64 {
		return DisplayName{}, ErrInvalidDisplayName
	}
	return DisplayName{value: trimmed}, nil
}

// String returns the canonical name.
func (d DisplayName) String() string { return d.value }

// User is the iam aggregate root.
type User struct {
	id           string
	email        EmailAddress
	password     PasswordHash
	displayName  DisplayName
	avatar       AvatarRef
	registeredAt time.Time
}

// NewUser builds a fresh user. Used by RegisterUser. Times are UTC.
func NewUser(
	id string,
	email EmailAddress,
	password PasswordHash,
	name DisplayName,
	registeredAt time.Time,
) (*User, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrUserNotFound
	}
	if email.IsZero() {
		return nil, ErrInvalidEmail
	}
	if password.IsZero() {
		return nil, ErrInvalidPasswordHash
	}
	if name.String() == "" {
		return nil, ErrInvalidDisplayName
	}
	return &User{
		id:           id,
		email:        email,
		password:     password,
		displayName:  name,
		avatar:       NoneAvatar,
		registeredAt: registeredAt.UTC(),
	}, nil
}

// HydrateUser rebuilds a user from persistence. Skips construction-time defaults
// like AvatarRef = none — that decision is the caller's.
func HydrateUser(
	id string,
	email EmailAddress,
	password PasswordHash,
	name DisplayName,
	avatar AvatarRef,
	registeredAt time.Time,
) *User {
	return &User{
		id:           id,
		email:        email,
		password:     password,
		displayName:  name,
		avatar:       avatar,
		registeredAt: registeredAt.UTC(),
	}
}

// ID accessor.
func (u *User) ID() string { return u.id }

// Email accessor.
func (u *User) Email() EmailAddress { return u.email }

// PasswordHash accessor — returns the stored hash for verification.
func (u *User) PasswordHash() PasswordHash { return u.password }

// DisplayName accessor.
func (u *User) DisplayName() DisplayName { return u.displayName }

// Avatar accessor.
func (u *User) Avatar() AvatarRef { return u.avatar }

// RegisteredAt accessor.
func (u *User) RegisteredAt() time.Time { return u.registeredAt }

// SetEmail mutates the email. Invariant 1–4.
func (u *User) SetEmail(e EmailAddress) error {
	if e.IsZero() {
		return ErrInvalidEmail
	}
	u.email = e
	return nil
}

// SetDisplayName mutates the display name. Invariant 6.
func (u *User) SetDisplayName(n DisplayName) error {
	if n.String() == "" {
		return ErrInvalidDisplayName
	}
	u.displayName = n
	return nil
}

// SetAvatar mutates the avatar reference.
func (u *User) SetAvatar(a AvatarRef) {
	u.avatar = a
}

// ChangePassword rotates the password hash. The caller is responsible for
// revoking refresh families (see invariant 7) by calling RefreshTokens.RevokeAllForUser
// in the same tx.
func (u *User) ChangePassword(newHash PasswordHash) error {
	if newHash.IsZero() {
		return ErrInvalidPasswordHash
	}
	u.password = newHash
	return nil
}
