package shared

import "strings"

// IdempotencyKey is a single-use opaque value attached to non-idempotent writes
// per ADR 0005. The empty string means "no key".
type IdempotencyKey string

// String returns the underlying value.
func (k IdempotencyKey) String() string { return string(k) }

// IsZero reports whether the key is empty.
func (k IdempotencyKey) IsZero() bool { return strings.TrimSpace(string(k)) == "" }
