package iam

import (
	"context"

	"github.com/micocards/api/internal/domain/iam"
)

// GetCurrentUser returns the logged-in user's profile.
type GetCurrentUser struct {
	Users iam.Users
}

// Handle resolves the user by id (carried in the JWT claims by the HTTP edge).
func (uc GetCurrentUser) Handle(ctx context.Context, in GetCurrentUserInput) (GetCurrentUserOutput, error) {
	if in.UserID == "" {
		return GetCurrentUserOutput{}, iam.ErrUnauthorized
	}
	u, err := uc.Users.ByID(ctx, in.UserID)
	if err != nil {
		return GetCurrentUserOutput{}, err
	}
	return GetCurrentUserOutput{User: ViewOf(u)}, nil
}
