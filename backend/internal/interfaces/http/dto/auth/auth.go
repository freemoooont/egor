// Package auth contains the HTTP DTOs for the iam context.
package auth

import "time"

// User mirrors the OpenAPI "User" schema.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"displayName"`
	AvatarRef    string    `json:"avatarRef"`
	RegisteredAt time.Time `json:"registeredAt"`
}

// AuthResponse is the response body for register/login/refresh.
type AuthResponse struct {
	AccessToken           string    `json:"accessToken"`
	RefreshToken          string    `json:"refreshToken"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
	User                  User      `json:"user"`
}

// RegisterRequest is the request body for POST /api/auth/register.
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

// LoginRequest is the request body for POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest is the request body for POST /api/auth/refresh and logout.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// RefreshResponse mirrors AuthResponse minus the user (refresh does not
// return the profile).
type RefreshResponse struct {
	AccessToken           string    `json:"accessToken"`
	RefreshToken          string    `json:"refreshToken"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
}

// UpdateProfileRequest is the body for PATCH/PUT /api/me.
type UpdateProfileRequest struct {
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
}

// ChangePasswordRequest is the body for POST /api/auth/change-password
// (a.k.a. POST /api/me/password).
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
