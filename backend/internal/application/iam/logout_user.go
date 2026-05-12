package iam

import (
	"context"
	"errors"

	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/domain/shared"
)

// LogoutUser revokes the family of the supplied refresh token.
type LogoutUser struct {
	RefreshTokens iam.RefreshTokens
	Hasher        RefreshHasher
	Clock         shared.Clock
	UoW           UnitOfWork
	Events        EventPublisher
}

// Handle revokes the family idempotently — repeated calls on an already
// revoked token return ok.
func (uc LogoutUser) Handle(ctx context.Context, in LogoutUserInput) (LogoutUserOutput, error) {
	if in.RefreshToken == "" {
		return LogoutUserOutput{}, iam.ErrRefreshTokenInvalid
	}
	hash := uc.Hasher.HashOpaque(in.RefreshToken)
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		tok, err := uc.RefreshTokens.ByOpaqueHash(ctx, hash)
		if err != nil {
			if errors.Is(err, iam.ErrRefreshTokenInvalid) {
				return nil // logout is idempotent — unknown token is "already logged out"
			}
			return err
		}
		now := uc.Clock.Now(ctx)
		if err := uc.RefreshTokens.RevokeFamily(ctx, tok.FamilyID, nil, iam.RevokeReasonLogout); err != nil {
			return err
		}
		return uc.Events.Publish(ctx, iam.RefreshTokenRevoked{
			UserID:    tok.UserID,
			FamilyID:  tok.FamilyID,
			Reason:    string(iam.RevokeReasonLogout),
			RevokedAt: now,
		})
	})
	if err != nil {
		return LogoutUserOutput{}, err
	}
	return LogoutUserOutput{OK: true}, nil
}
