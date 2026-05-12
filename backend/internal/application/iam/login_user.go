package iam

import (
	"context"
	"errors"

	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/domain/shared"
)

// LoginUser authenticates a user against stored credentials and mints a fresh
// access+refresh pair.
type LoginUser struct {
	Users         iam.Users
	RefreshTokens iam.RefreshTokens
	Hasher        PasswordHasher
	IDs           shared.IDGenerator
	Clock         shared.Clock
	Tokens        RefreshTokenMinter
	AccessSigner  AccessTokenSigner
	UoW           UnitOfWork
	Events        EventPublisher
}

// Handle resolves the user by email, verifies the password, and mints tokens.
func (uc LoginUser) Handle(ctx context.Context, in LoginUserInput) (LoginUserOutput, error) {
	email, err := iam.NewEmailAddress(in.Email)
	if err != nil {
		return LoginUserOutput{}, err
	}

	var out LoginUserOutput
	err = uc.UoW.Do(ctx, func(ctx context.Context) error {
		user, err := uc.Users.ByEmail(ctx, email)
		if err != nil {
			if errors.Is(err, iam.ErrUserNotFound) {
				return iam.ErrInvalidCredentials
			}
			return err
		}
		if err := uc.Hasher.Compare(ctx, user.PasswordHash(), in.Password); err != nil {
			return iam.ErrInvalidCredentials
		}

		now := uc.Clock.Now(ctx)
		bundle, refIssued, err := mintAuthBundle(ctx, uc.AccessSigner, uc.Tokens, uc.RefreshTokens, uc.IDs, now, user.ID())
		if err != nil {
			return err
		}

		out = LoginUserOutput{User: ViewOf(user), Auth: bundle}
		return uc.Events.Publish(ctx,
			iam.UserLoggedIn{UserID: user.ID(), LoggedInAt: now},
			refIssued,
		)
	})
	if err != nil {
		return LoginUserOutput{}, err
	}
	return out, nil
}
