package iam

import (
	"context"

	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/domain/shared"
)

// ChangePassword rotates the password and revokes every active refresh family
// for the user (invariant 7).
type ChangePassword struct {
	Users         iam.Users
	RefreshTokens iam.RefreshTokens
	Hasher        PasswordHasher
	Clock         shared.Clock
	UoW           UnitOfWork
	Events        EventPublisher
}

// Handle verifies the current password, hashes the new one, and revokes
// refresh families.
func (uc ChangePassword) Handle(ctx context.Context, in ChangePasswordInput) (ChangePasswordOutput, error) {
	if in.UserID == "" {
		return ChangePasswordOutput{}, iam.ErrUnauthorized
	}
	if err := uc.Hasher.Strength(in.NewPassword); err != nil {
		return ChangePasswordOutput{}, err
	}

	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		u, err := uc.Users.ByID(ctx, in.UserID)
		if err != nil {
			return err
		}
		if err := uc.Hasher.Compare(ctx, u.PasswordHash(), in.CurrentPassword); err != nil {
			return iam.ErrInvalidCredentials
		}
		newHash, err := uc.Hasher.Hash(ctx, in.NewPassword)
		if err != nil {
			return err
		}
		if err := u.ChangePassword(newHash); err != nil {
			return err
		}
		if err := uc.Users.Save(ctx, u); err != nil {
			return err
		}
		now := uc.Clock.Now(ctx)
		if err := uc.RefreshTokens.RevokeAllForUser(ctx, u.ID(), nil, iam.RevokeReasonPasswordChange); err != nil {
			return err
		}
		return uc.Events.Publish(ctx, iam.RefreshTokenRevoked{
			UserID:    u.ID(),
			Reason:    string(iam.RevokeReasonPasswordChange),
			RevokedAt: now,
		})
	})
	if err != nil {
		return ChangePasswordOutput{}, err
	}
	return ChangePasswordOutput{OK: true}, nil
}
