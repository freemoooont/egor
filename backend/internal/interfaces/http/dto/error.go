// Package dto holds the wire shapes for the HTTP edge. The error envelope
// matches the contract documented in ADR 0006: {error, message?, details?}.
package dto

// ErrorEnvelope is the canonical JSON error body.
type ErrorEnvelope struct {
	Error   string         `json:"error"`
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// OK is the trivial success envelope used by ChangePassword/Logout/Delete.
type OK struct {
	OK bool `json:"ok"`
}
