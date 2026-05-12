package iam

import (
	"context"
	"time"

	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/domain/shared"
)

// RegisterUser is the registration use case. See use-cases.md → RegisterUser.
type RegisterUser struct {
	Users          iam.Users
	RefreshTokens  iam.RefreshTokens
	Hasher         PasswordHasher
	IDs            shared.IDGenerator
	Clock          shared.Clock
	Tokens         RefreshTokenMinter
	AccessSigner   AccessTokenSigner
	UoW            UnitOfWork
	Events         EventPublisher
}

// Handle creates a new user, mints the initial access+refresh tokens, and
// publishes UserRegistered + RefreshTokenIssued.
func (uc RegisterUser) Handle(ctx context.Context, in RegisterUserInput) (RegisterUserOutput, error) {
	email, err := iam.NewEmailAddress(in.Email)
	if err != nil {
		return RegisterUserOutput{}, err
	}
	name, err := iam.NewDisplayName(in.DisplayName)
	if err != nil {
		return RegisterUserOutput{}, err
	}
	if err := uc.Hasher.Strength(in.Password); err != nil {
		return RegisterUserOutput{}, err
	}

	var out RegisterUserOutput
	err = uc.UoW.Do(ctx, func(ctx context.Context) error {
		exists, err := uc.Users.EmailExists(ctx, email)
		if err != nil {
			return err
		}
		if exists {
			return iam.ErrEmailTaken
		}

		hash, err := uc.Hasher.Hash(ctx, in.Password)
		if err != nil {
			return err
		}

		now := uc.Clock.Now(ctx)
		userID := uc.IDs.NewID(ctx)
		user, err := iam.NewUser(userID, email, hash, name, now)
		if err != nil {
			return err
		}
		if err := uc.Users.Save(ctx, user); err != nil {
			return err
		}

		bundle, refIssued, err := mintAuthBundle(ctx, uc.AccessSigner, uc.Tokens, uc.RefreshTokens, uc.IDs, now, user.ID())
		if err != nil {
			return err
		}

		out = RegisterUserOutput{
			User: ViewOf(user),
			Auth: bundle,
		}

		return uc.Events.Publish(ctx,
			iam.UserRegistered{
				UserID:       user.ID(),
				Email:        user.Email().String(),
				DisplayName:  user.DisplayName().String(),
				RegisteredAt: now,
			},
			refIssued,
		)
	})
	if err != nil {
		return RegisterUserOutput{}, err
	}
	return out, nil
}

// mintAuthBundle issues a fresh access+refresh pair, persists the refresh row,
// and returns the matching domain event. Reused by Login and Refresh.
func mintAuthBundle(
	ctx context.Context,
	signer AccessTokenSigner,
	mint RefreshTokenMinter,
	repo iam.RefreshTokens,
	ids shared.IDGenerator,
	now time.Time,
	userID string,
) (AuthBundle, iam.RefreshTokenIssued, error) {
	access, accessExpUnix, err := signer.SignAccessToken(ctx, userID)
	if err != nil {
		return AuthBundle{}, iam.RefreshTokenIssued{}, err
	}
	refreshPlain, refreshHash, err := mint.Mint(ctx)
	if err != nil {
		return AuthBundle{}, iam.RefreshTokenIssued{}, err
	}
	tokenID := ids.NewID(ctx)
	familyID := ids.NewID(ctx)
	expires := now.Add(time.Duration(RefreshTokenTTLSeconds) * time.Second)
	tok := iam.RefreshToken{
		ID:         tokenID,
		FamilyID:   familyID,
		UserID:     userID,
		OpaqueHash: refreshHash,
		IssuedAt:   now,
		ExpiresAt:  expires,
	}
	if err := repo.Save(ctx, tok); err != nil {
		return AuthBundle{}, iam.RefreshTokenIssued{}, err
	}
	bundle := AuthBundle{
		AccessToken:           access,
		RefreshToken:          refreshPlain,
		AccessTokenExpiresAt:  time.Unix(accessExpUnix, 0).UTC(),
		RefreshTokenExpiresAt: expires,
	}
	ev := iam.RefreshTokenIssued{
		UserID:    userID,
		FamilyID:  familyID,
		TokenID:   tokenID,
		IssuedAt:  now,
		ExpiresAt: expires,
	}
	return bundle, ev, nil
}
