// Package iam implements the identity-and-access bounded context: users,
// credentials, refresh-token families. Pure domain — no infra imports.
package iam

import "errors"

// Domain sentinel errors. The HTTP edge maps these to status codes via the
// table in ADR 0006.
var (
	ErrInvalidEmail         = errors.New("iam: invalid email")
	ErrInvalidDisplayName   = errors.New("iam: invalid display name")
	ErrInvalidPasswordHash  = errors.New("iam: invalid password hash")
	ErrPasswordTooWeak      = errors.New("iam: password too weak")
	ErrEmailTaken           = errors.New("iam: email taken")
	ErrUserNotFound         = errors.New("iam: user not found")
	ErrInvalidCredentials   = errors.New("iam: invalid credentials")
	ErrUnauthorized         = errors.New("iam: unauthorized")
	ErrRefreshTokenInvalid  = errors.New("iam: refresh token invalid")
	ErrRefreshTokenExpired  = errors.New("iam: refresh token expired")
	ErrRefreshTokenReused   = errors.New("iam: refresh token reused")
	ErrIdempotencyConflict  = errors.New("iam: idempotency key conflict")
	ErrIdempotencyKeyNeeded = errors.New("iam: idempotency key required")
)
