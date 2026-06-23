package slogutil

import (
	"context"
	"fmt"
	"log/slog"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// ─── go-kit → slog bridge ────────────────────────────────────────────────────

// GoKitHandler is a slog.Handler that forwards every log record to a
// go-kit/log Logger. It translates slog levels to go-kit level values and
// flattens slog attribute groups into dotted key names so that go-kit's
// key=value output remains human-readable.
//
// Use FromGoKitLogger as a convenience constructor.
type GoKitHandler struct {
	logger   kitlog.Logger
	preKVs   []interface{} // key-value pairs accumulated by WithAttrs
	groupPfx string        // current group prefix for Handle's attrs
}

// NewGoKitHandler creates a GoKitHandler that writes to logger.
func NewGoKitHandler(logger kitlog.Logger) *GoKitHandler {
	return &GoKitHandler{logger: logger}
}

// FromGoKitLogger wraps a go-kit Logger and returns a *slog.Logger that
// forwards all records through it. This is the primary way to bring slog
// into an existing component that already owns a go-kit logger.
func FromGoKitLogger(logger kitlog.Logger) *slog.Logger {
	return slog.New(NewGoKitHandler(logger))
}

// Enabled implements slog.Handler. GoKitHandler always returns true because
// level filtering is expected to be handled by the wrapped go-kit logger.
func (h *GoKitHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle implements slog.Handler. It converts the slog.Record into a flat
// sequence of key-value pairs and forwards them to the underlying go-kit
// logger with an appropriate level prefix.
func (h *GoKitHandler) Handle(_ context.Context, r slog.Record) error {
	// Apply go-kit level wrapper.
	var logger kitlog.Logger
	switch {
	case r.Level >= slog.LevelError:
		logger = level.Error(h.logger)
	case r.Level >= slog.LevelWarn:
		logger = level.Warn(h.logger)
	case r.Level >= slog.LevelInfo:
		logger = level.Info(h.logger)
	default:
		logger = level.Debug(h.logger)
	}

	// Build the flat key-value slice.
	kvs := make([]interface{}, 0, 2+len(h.preKVs)+r.NumAttrs()*2)
	kvs = append(kvs, "msg", r.Message)
	kvs = append(kvs, h.preKVs...)

	r.Attrs(func(a slog.Attr) bool {
		kvs = appendAttr(kvs, h.groupPfx, a)
		return true
	})

	return logger.Log(kvs...)
}

// WithAttrs implements slog.Handler. The provided attrs are resolved
// immediately (with the current group prefix) and stored for inclusion in
// every subsequent record.
func (h *GoKitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	kvs := make([]interface{}, len(h.preKVs), len(h.preKVs)+len(attrs)*2)
	copy(kvs, h.preKVs)
	for _, a := range attrs {
		kvs = appendAttr(kvs, h.groupPfx, a)
	}
	return &GoKitHandler{
		logger:   h.logger,
		preKVs:   kvs,
		groupPfx: h.groupPfx,
	}
}

// WithGroup implements slog.Handler. Subsequent attrs (both from WithAttrs
// and from individual log calls) will be prefixed with "name.".
func (h *GoKitHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	pfx := h.groupPfx + name + "."
	return &GoKitHandler{
		logger:   h.logger,
		preKVs:   h.preKVs,
		groupPfx: pfx,
	}
}

// ─── slog → go-kit bridge ────────────────────────────────────────────────────

// GoKitAdapter wraps a *slog.Logger and satisfies the go-kit/log Logger
// interface. It lets existing components that accept a kitlog.Logger receive a
// slog-backed implementation without any changes to their call sites.
//
// The adapter understands go-kit level values produced by
// github.com/go-kit/kit/log/level so that calls such as
//
//	level.Error(adapter).Log("msg", "something went wrong")
//
// are forwarded at the correct slog level.
type GoKitAdapter struct {
	logger *slog.Logger
}

// NewGoKitAdapter wraps logger so it satisfies kitlog.Logger.
func NewGoKitAdapter(logger *slog.Logger) *GoKitAdapter {
	return &GoKitAdapter{logger: logger}
}

// Log implements kitlog.Logger. It accepts go-kit-style variadic key-value
// pairs. The "level" key (if present and carrying a go-kit level.Value) is
// translated to the corresponding slog level; the "msg" key becomes the slog
// message; all remaining pairs become slog attributes.
func (a *GoKitAdapter) Log(kvs ...interface{}) error {
	if len(kvs) == 0 {
		return nil
	}

	slogLevel := slog.LevelInfo
	msg := ""
	args := make([]interface{}, 0, len(kvs))

	for i := 0; i+1 < len(kvs); i += 2 {
		k := fmt.Sprint(kvs[i])
		v := kvs[i+1]

		switch k {
		case "level":
			if lv, ok := v.(level.Value); ok {
				slogLevel = gokitLevelToSlog(lv)
			}
			// Always skip the level kv so it is not double-emitted.
		case "msg":
			msg = fmt.Sprint(v)
		default:
			args = append(args, k, v)
		}
	}

	// Guard against an odd-length slice (go-kit appends ErrMissingValue for
	// missing values, but be defensive here too).
	if len(kvs)%2 != 0 {
		args = append(args, "MISSING", kvs[len(kvs)-1])
	}

	a.logger.Log(context.Background(), slogLevel, msg, args...)
	return nil
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// gokitLevelToSlog maps a go-kit level.Value to the corresponding slog.Level.
func gokitLevelToSlog(v level.Value) slog.Level {
	switch v.String() {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// appendAttr converts a slog.Attr into one or more key-value pairs and
// appends them to kvs. Group-kind attrs are flattened recursively using a
// dotted prefix derived from the group name.
func appendAttr(kvs []interface{}, prefix string, a slog.Attr) []interface{} {
	// Resolve any LogValuer before inspecting kind.
	a.Value = a.Value.Resolve()

	// Skip zero-value attrs (e.g. slog.Attr{}).
	if a.Equal(slog.Attr{}) {
		return kvs
	}

	if a.Value.Kind() == slog.KindGroup {
		// Flatten by extending the dotted prefix.
		// prefix is either "" or already ends with ".", so plain concatenation
		// produces the correct "parent.child." string.
		subPrefix := prefix
		if a.Key != "" {
			subPrefix = prefix + a.Key + "."
		}
		for _, sub := range a.Value.Group() {
			kvs = appendAttr(kvs, subPrefix, sub)
		}
		return kvs
	}

	key := prefix + a.Key
	return append(kvs, key, a.Value.Any())
}
