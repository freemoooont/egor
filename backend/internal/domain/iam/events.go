package iam

import "time"

// Event is the marker interface every domain event implements.
type Event interface {
	Name() string
}

// UserRegistered is emitted on RegisterUser. See domain-events.md.
type UserRegistered struct {
	UserID       string
	Email        string
	DisplayName  string
	RegisteredAt time.Time
}

// Name returns the wire name.
func (UserRegistered) Name() string { return "iam.UserRegistered" }

// UserLoggedIn is emitted on LoginUser.
type UserLoggedIn struct {
	UserID     string
	LoggedInAt time.Time
	UserAgent  string
	IP         string
}

// Name returns the wire name.
func (UserLoggedIn) Name() string { return "iam.UserLoggedIn" }

// RefreshTokenIssued is emitted on LoginUser and RefreshAccessToken.
type RefreshTokenIssued struct {
	UserID    string
	FamilyID  string
	TokenID   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// Name returns the wire name.
func (RefreshTokenIssued) Name() string { return "iam.RefreshTokenIssued" }

// RefreshTokenRevoked is emitted on rotation, logout, password-change, or reuse
// detection.
type RefreshTokenRevoked struct {
	UserID    string
	FamilyID  string
	TokenID   string
	Reason    string
	RevokedAt time.Time
}

// Name returns the wire name.
func (RefreshTokenRevoked) Name() string { return "iam.RefreshTokenRevoked" }
