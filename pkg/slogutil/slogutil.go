// Package slogutil provides helpers for structured logging using the standard
// library's log/slog package. It is designed to work alongside the go-kit/log
// interface that is used throughout the rest of Loki; the bridge types in
// bridge.go let new code consume the slog API while the underlying log
// transport remains go-kit.
//
// Typical usage for new code:
//
//	logger := slogutil.NewLogger(slogutil.Options{
//	    Level:  slog.LevelDebug,
//	    Format: slogutil.FormatJSON,
//	})
//	logger = slogutil.WithComponent(logger, "ingester")
//	logger.Info("starting", "addr", addr)
//
// Bridging existing go-kit loggers:
//
//	// Wrap a go-kit logger so it is usable as *slog.Logger.
//	slogLogger := slogutil.FromGoKitLogger(gokitLogger)
//
//	// Wrap a *slog.Logger so it satisfies go-kit's Logger interface.
//	gokitCompat := slogutil.NewGoKitAdapter(slogLogger)
package slogutil

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Format represents the log output format.
type Format string

const (
	// FormatLogfmt is the logfmt / text output format (default).
	FormatLogfmt Format = "logfmt"
	// FormatJSON is the JSON output format.
	FormatJSON Format = "json"
)

// Options configures the logger created by NewLogger.
type Options struct {
	// Level is the minimum log level. Defaults to slog.LevelInfo.
	Level slog.Level
	// Format is the output format. Defaults to FormatLogfmt.
	Format Format
	// Writer is the output destination. Defaults to os.Stderr.
	Writer io.Writer
	// AddSource adds the source file and line number to each log record.
	AddSource bool
}

// NewLogger creates a new *slog.Logger from the given options.
// When no Writer is supplied it writes to os.Stderr. When no Format is
// supplied it uses logfmt (slog.TextHandler).
func NewLogger(opts Options) *slog.Logger {
	w := opts.Writer
	if w == nil {
		w = os.Stderr
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     opts.Level,
		AddSource: opts.AddSource,
	}

	var h slog.Handler
	if opts.Format == FormatJSON {
		h = slog.NewJSONHandler(w, handlerOpts)
	} else {
		h = slog.NewTextHandler(w, handlerOpts)
	}
	return slog.New(h)
}

// Nop returns a *slog.Logger that silently discards every record.
// It is useful in tests and as a safe zero-value alternative to nil.
func Nop() *slog.Logger {
	return slog.New(nopHandler{})
}

// WithComponent returns a new logger that appends a "component" attribute to
// every record it emits. This follows the Loki convention of tagging log lines
// with the subsystem that produced them.
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	return logger.With("component", component)
}

// WithNamespace returns a new logger whose subsequent attributes are nested
// under the given slog group name. All attributes added after this call (either
// via With or inline in logging calls) will be scoped to that namespace.
func WithNamespace(logger *slog.Logger, namespace string) *slog.Logger {
	return logger.WithGroup(namespace)
}

// ParseLevel converts a string level name ("debug", "info", "warn", "error")
// into its slog.Level constant. It returns an error for unrecognised strings.
func ParseLevel(s string) (slog.Level, error) {
	switch s {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf(
			"slogutil: unknown log level %q; valid values are debug, info, warn, error", s,
		)
	}
}

// nopHandler is a slog.Handler that discards every record.
type nopHandler struct{}

func (nopHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (nopHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (h nopHandler) WithAttrs(_ []slog.Attr) slog.Handler        { return h }
func (h nopHandler) WithGroup(_ string) slog.Handler             { return h }
