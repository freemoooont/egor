package iamrepo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	domiam "github.com/micocards/api/internal/domain/iam"
)

// rowScanner is satisfied by both pgx.Row and pgx.Rows (Scan is the only common
// method we use here).
type rowScanner interface {
	Scan(dest ...any) error
}

// scanUser turns a fetched row into a fully hydrated domain User.
func scanUser(r rowScanner) (*domiam.User, error) {
	var (
		id, email, hash, name, kind, ref string
		registeredAt                     time.Time
	)
	if err := r.Scan(&id, &email, &hash, &name, &kind, &ref, &registeredAt); err != nil {
		return nil, err
	}
	emailVO, err := domiam.NewEmailAddress(email)
	if err != nil {
		return nil, fmt.Errorf("scanUser email: %w", err)
	}
	hashVO, err := domiam.NewPasswordHash(hash)
	if err != nil {
		return nil, fmt.Errorf("scanUser password hash: %w", err)
	}
	nameVO, err := domiam.NewDisplayName(name)
	if err != nil {
		return nil, fmt.Errorf("scanUser display name: %w", err)
	}
	avatar := domiam.AvatarRef{Kind: domiam.AvatarRefKind(kind), Ref: ref}
	if avatar.Kind == "" {
		avatar = domiam.NoneAvatar
	}
	return domiam.HydrateUser(id, emailVO, hashVO, nameVO, avatar, registeredAt), nil
}

// scanRefreshToken turns a fetched row into a domain RefreshToken.
func scanRefreshToken(r rowScanner) (domiam.RefreshToken, error) {
	var (
		id, familyID, userID, parent, hash, note string
		issued, expires                          time.Time
		revoked                                  *time.Time
	)
	if err := r.Scan(&id, &familyID, &userID, &parent, &hash, &issued, &expires, &revoked, &note); err != nil {
		return domiam.RefreshToken{}, err
	}
	out := domiam.RefreshToken{
		ID:         id,
		FamilyID:   familyID,
		UserID:     userID,
		ParentID:   parent,
		OpaqueHash: hash,
		IssuedAt:   issued,
		ExpiresAt:  expires,
		RevokeNote: domiam.RevokeReason(note),
	}
	if revoked != nil {
		t := revoked.UTC()
		out.RevokedAt = &t
	}
	return out, nil
}

// isUniqueViolation reports whether err is a pgconn.PgError with SQLSTATE
// 23505 — and, if constraint is non-empty, whether that specific constraint
// was violated. Returns false on any non-pg error.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if pgErr.Code != "23505" {
		return false
	}
	if constraint == "" {
		return true
	}
	// pg uses both "<table>_<col>_key" and "<table>_<col>_idx" for unique
	// constraints; tolerate either by substring matching.
	return strings.Contains(pgErr.ConstraintName, constraint) ||
		strings.Contains(pgErr.Message, constraint)
}
