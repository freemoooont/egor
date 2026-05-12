// Package logger wraps log/slog with a small builder that selects JSON in
// production and text in development based on the LOG_FORMAT env var.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Format selects the slog handler implementation.
type Format string

const (
	// FormatJSON emits JSON to stdout (production).
	FormatJSON Format = "json"
	// FormatText emits human-readable lines to stderr (development).
	FormatText Format = "text"
)

// Options control New.
type Options struct {
	Format Format
	Level  slog.Level
	Writer io.Writer // defaults to os.Stdout (json) / os.Stderr (text)
}

// FromEnv reads LOG_FORMAT and LOG_LEVEL. LOG_FORMAT defaults to "text" in dev
// builds (-tags=debug) but to "json" by default — matching the docs/stack.md
// guidance "slog JSON to stdout".
func FromEnv() Options {
	format := Format(strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT"))))
	if format != FormatText && format != FormatJSON {
		format = FormatJSON
	}
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return Options{Format: format, Level: level}
}

// New returns a slog.Logger configured per opts.
func New(opts Options) *slog.Logger {
	w := opts.Writer
	hOpts := &slog.HandlerOptions{Level: opts.Level}
	switch opts.Format {
	case FormatText:
		if w == nil {
			w = os.Stderr
		}
		return slog.New(slog.NewTextHandler(w, hOpts))
	default:
		if w == nil {
			w = os.Stdout
		}
		return slog.New(slog.NewJSONHandler(w, hOpts))
	}
}
