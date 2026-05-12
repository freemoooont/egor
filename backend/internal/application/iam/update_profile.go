package iam

import (
	"context"
	"errors"

	"github.com/micocards/api/internal/domain/iam"
)

// UpdateProfile mutates the user's display name and/or email.
type UpdateProfile struct {
	Users iam.Users
	UoW   UnitOfWork
}

// Handle validates and applies the patch.
func (uc UpdateProfile) Handle(ctx context.Context, in UpdateProfileInput) (UpdateProfileOutput, error) {
	if in.UserID == "" {
		return UpdateProfileOutput{}, iam.ErrUnauthorized
	}

	var out UpdateProfileOutput
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		u, err := uc.Users.ByID(ctx, in.UserID)
		if err != nil {
			return err
		}
		if in.DisplayName != nil {
			name, err := iam.NewDisplayName(*in.DisplayName)
			if err != nil {
				return err
			}
			if err := u.SetDisplayName(name); err != nil {
				return err
			}
		}
		if in.Email != nil {
			email, err := iam.NewEmailAddress(*in.Email)
			if err != nil {
				return err
			}
			if email.String() != u.Email().String() {
				exists, err := uc.Users.EmailExists(ctx, email)
				if err != nil {
					return err
				}
				if exists {
					return iam.ErrEmailTaken
				}
				if err := u.SetEmail(email); err != nil {
					return err
				}
			}
		}
		if err := uc.Users.Save(ctx, u); err != nil {
			if errors.Is(err, iam.ErrEmailTaken) {
				return iam.ErrEmailTaken
			}
			return err
		}
		out = UpdateProfileOutput{User: ViewOf(u)}
		return nil
	})
	if err != nil {
		return UpdateProfileOutput{}, err
	}
	return out, nil
}
