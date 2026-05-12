package iam

import (
	"regexp"
	"strings"
)

// EmailAddress is a validated, lower-cased email value object.
type EmailAddress struct {
	value string
}

// rfc5321ish is a pragmatic RFC-5321 subset matcher. The full grammar is
// notoriously over-permissive in practice; this matches what users actually
// type and what our pgx layer accepts as TEXT.
var rfc5321ish = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)

// NewEmailAddress trims, lower-cases, validates, and length-checks the input.
// Invariants: User_EmailAddressMustBeNonEmptyAfterTrim,
// User_EmailAddressMustBeRFC5321Valid, User_EmailAddressIsLowerCasedOnConstruction.
func NewEmailAddress(raw string) (EmailAddress, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return EmailAddress{}, ErrInvalidEmail
	}
	if len(trimmed) > 254 {
		return EmailAddress{}, ErrInvalidEmail
	}
	lower := strings.ToLower(trimmed)
	if !rfc5321ish.MatchString(lower) {
		return EmailAddress{}, ErrInvalidEmail
	}
	return EmailAddress{value: lower}, nil
}

// String returns the canonical lower-cased value.
func (e EmailAddress) String() string { return e.value }

// IsZero reports whether this is the zero-value email.
func (e EmailAddress) IsZero() bool { return e.value == "" }
