package iam

import (
	"time"

	"github.com/micocards/api/internal/domain/iam"
)

// UserView is the application-layer projection of an iam.User.
type UserView struct {
	ID           string
	Email        string
	DisplayName  string
	AvatarKind   iam.AvatarRefKind
	AvatarRef    string
	RegisteredAt time.Time
}

// ViewOf builds a UserView from a domain user.
func ViewOf(u *iam.User) UserView {
	return UserView{
		ID:           u.ID(),
		Email:        u.Email().String(),
		DisplayName:  u.DisplayName().String(),
		AvatarKind:   u.Avatar().Kind,
		AvatarRef:    u.Avatar().Ref,
		RegisteredAt: u.RegisteredAt(),
	}
}

// AuthBundle is what register/login/refresh return — paired access+refresh
// tokens plus the matching expiry stamps.
type AuthBundle struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
}

// RegisterUserInput / Output.
type RegisterUserInput struct {
	Email       string
	Password    string
	DisplayName string
}

// RegisterUserOutput is the response payload.
type RegisterUserOutput struct {
	User UserView
	Auth AuthBundle
}

// LoginUserInput / Output.
type LoginUserInput struct {
	Email    string
	Password string
}

// LoginUserOutput is the response payload.
type LoginUserOutput struct {
	User UserView
	Auth AuthBundle
}

// RefreshAccessTokenInput / Output.
type RefreshAccessTokenInput struct {
	RefreshToken string
}

// RefreshAccessTokenOutput is the response payload.
type RefreshAccessTokenOutput struct {
	Auth AuthBundle
}

// LogoutUserInput / Output.
type LogoutUserInput struct {
	RefreshToken string
}

// LogoutUserOutput is the response payload.
type LogoutUserOutput struct {
	OK bool
}

// GetCurrentUserInput / Output.
type GetCurrentUserInput struct {
	UserID string
}

// GetCurrentUserOutput is the response payload.
type GetCurrentUserOutput struct {
	User UserView
}

// UpdateProfileInput / Output.
type UpdateProfileInput struct {
	UserID      string
	DisplayName *string
	Email       *string
}

// UpdateProfileOutput is the response payload.
type UpdateProfileOutput struct {
	User UserView
}

// ChangePasswordInput / Output.
type ChangePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

// ChangePasswordOutput is the response payload.
type ChangePasswordOutput struct {
	OK bool
}
