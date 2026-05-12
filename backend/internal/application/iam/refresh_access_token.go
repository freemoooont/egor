package iam

import (
	"context"
	"errors"

	"github.com/micocards/api/internal/domain/iam"
	"github.com/micocards/api/internal/domain/shared"
)

// RefreshAccessToken rotates a refresh token in place. Reuse of an
// already-revoked token revokes the entire family (ADR 0003 / invariant 9).
type RefreshAccessToken struct {
	RefreshTokens iam.RefreshTokens
	IDs           shared.IDGenerator
	Clock         shared.Clock
	Tokens        RefreshTokenMinter
	AccessSigner  AccessTokenSigner
	UoW           UnitOfWork
	Events        EventPublisher
	Hasher        RefreshHasher // hashes the inbound plaintext for lookup
}

// RefreshHasher hashes the inbound opaque refresh value to compare against the
// stored hash. Implementations live in internal/infrastructure/auth.
type RefreshHasher interface {
	HashOpaque(plaintext string) string
}

// Handle rotates the supplied refresh token.
func (uc RefreshAccessToken) Handle(ctx context.Context, in RefreshAccessTokenInput) (RefreshAccessTokenOutput, error) {
	if in.RefreshToken == "" {
		return RefreshAccessTokenOutput{}, iam.ErrRefreshTokenInvalid
	}
	hash := uc.Hasher.HashOpaque(in.RefreshToken)

	var out RefreshAccessTokenOutput
	err := uc.UoW.Do(ctx, func(ctx context.Context) error {
		consumed, err := uc.RefreshTokens.ByOpaqueHash(ctx, hash)
		if err != nil {
			if errors.Is(err, iam.ErrRefreshTokenInvalid) {
				return iam.ErrRefreshTokenInvalid
			}
			return err
		}
		now := uc.Clock.Now(ctx)
		if consumed.IsExpired(now) {
			return iam.ErrRefreshTokenExpired
		}
		if consumed.IsRevoked() {
			// reuse detected — kill the entire family
			_ = uc.RefreshTokens.RevokeFamily(ctx, consumed.FamilyID, nil, iam.RevokeReasonReuseDetected)
			_ = uc.Events.Publish(ctx, iam.RefreshTokenRevoked{
				UserID:    consumed.UserID,
				FamilyID:  consumed.FamilyID,
				Reason:    string(iam.RevokeReasonReuseDetected),
				RevokedAt: now,
			})
			return iam.ErrRefreshTokenReused
		}

		// Revoke the consumed token (rotation reason).
		if err := uc.RefreshTokens.RevokeOne(ctx, consumed.ID, nil, iam.RevokeReasonRotation); err != nil {
			return err
		}

		// Mint the rotated pair.
		access, accessExp, err := uc.AccessSigner.SignAccessToken(ctx, consumed.UserID)
		if err != nil {
			return err
		}
		newPlain, newHash, err := uc.Tokens.Mint(ctx)
		if err != nil {
			return err
		}
		newID := uc.IDs.NewID(ctx)
		ttl := consumed.ExpiresAt.Sub(consumed.IssuedAt)
		exp := now.Add(ttl)
		newTok := iam.RefreshToken{
			ID:         newID,
			FamilyID:   consumed.FamilyID,
			UserID:     consumed.UserID,
			ParentID:   consumed.ID,
			OpaqueHash: newHash,
			IssuedAt:   now,
			ExpiresAt:  exp,
		}
		if err := uc.RefreshTokens.Save(ctx, newTok); err != nil {
			return err
		}

		out = RefreshAccessTokenOutput{Auth: AuthBundle{
			AccessToken:           access,
			RefreshToken:          newPlain,
			AccessTokenExpiresAt:  unixToTime(accessExp),
			RefreshTokenExpiresAt: exp,
		}}

		return uc.Events.Publish(ctx,
			iam.RefreshTokenRevoked{
				UserID:    consumed.UserID,
				FamilyID:  consumed.FamilyID,
				TokenID:   consumed.ID,
				Reason:    string(iam.RevokeReasonRotation),
				RevokedAt: now,
			},
			iam.RefreshTokenIssued{
				UserID:    consumed.UserID,
				FamilyID:  consumed.FamilyID,
				TokenID:   newID,
				IssuedAt:  now,
				ExpiresAt: exp,
			},
		)
	})
	if err != nil {
		return RefreshAccessTokenOutput{}, err
	}
	return out, nil
}
