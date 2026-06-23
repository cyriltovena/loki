package slogutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/grafana/loki/pkg/slogutil"
)

// ─── NewLogger ────────────────────────────────────────────────────────────────

func TestNewLogger_LogfmtOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := slogutil.NewLogger(slogutil.Options{
		Writer: &buf,
		Level:  slog.LevelDebug,
	})

	logger.Info("hello world", "key", "value")

	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Fatalf("expected message in output, got: %s", out)
	}
	if !strings.Contains(out, "key=value") {
		t.Fatalf("expected key=value in output, got: %s", out)
	}
}

func TestNewLogger_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := slogutil.NewLogger(slogutil.Options{
		Writer:  &buf,
		Format:  slogutil.FormatJSON,
		Level:   slog.LevelDebug,
	})

	logger.Info("hello world", "key", "value")

	m := mustParseJSON(t, buf.Bytes())
	if m["msg"] != "hello world" {
		t.Errorf("unexpected msg: %v", m["msg"])
	}
	if m["key"] != "value" {
		t.Errorf("unexpected key: %v", m["key"])
	}
}

func TestNewLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := slogutil.NewLogger(slogutil.Options{
		Writer: &buf,
		Format: slogutil.FormatJSON,
		Level:  slog.LevelInfo,
	})

	logger.Debug("this should be dropped")
	logger.Info("this should appear")

	out := buf.String()
	if strings.Contains(out, "dropped") {
		t.Errorf("debug message should have been filtered, got: %s", out)
	}
	if !strings.Contains(out, "appear") {
		t.Errorf("info message should appear in output, got: %s", out)
	}
}

// ─── Nop ──────────────────────────────────────────────────────────────────────

func TestNop_DoesNotPanic(t *testing.T) {
	logger := slogutil.Nop()
	logger.Debug("debug", "k", "v")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error", "err", "boom")
	logger = slogutil.WithComponent(logger, "test")
	logger.Info("with component")
}

// ─── WithComponent ────────────────────────────────────────────────────────────

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	logger := slogutil.NewLogger(slogutil.Options{
		Writer: &buf,
		Format: slogutil.FormatJSON,
	})
	logger = slogutil.WithComponent(logger, "ingester")
	logger.Info("started")

	m := mustParseJSON(t, buf.Bytes())
	if m["component"] != "ingester" {
		t.Errorf("expected component=ingester, got: %v", m["component"])
	}
}

// ─── WithNamespace ────────────────────────────────────────────────────────────

func TestWithNamespace(t *testing.T) {
	var buf bytes.Buffer
	logger := slogutil.NewLogger(slogutil.Options{
		Writer: &buf,
		Format: slogutil.FormatJSON,
	})
	logger = slogutil.WithNamespace(logger, "storage")
	logger.Info("init", "disk", "ssd")

	m := mustParseJSON(t, buf.Bytes())
	ns, ok := m["storage"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested storage group, got top-level keys: %v", m)
	}
	if ns["disk"] != "ssd" {
		t.Errorf("expected storage.disk=ssd, got: %v", ns)
	}
}

// ─── ParseLevel ───────────────────────────────────────────────────────────────

func TestParseLevel(t *testing.T) {
	cases := []struct {
		input   string
		want    slog.Level
		wantErr bool
	}{
		{"debug", slog.LevelDebug, false},
		{"info", slog.LevelInfo, false},
		{"warn", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"ERROR", slog.LevelInfo, true}, // case-sensitive
		{"trace", slog.LevelInfo, true},
		{"", slog.LevelInfo, true},
	}
	for _, c := range cases {
		got, err := slogutil.ParseLevel(c.input)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseLevel(%q): expected error, got nil", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseLevel(%q): unexpected error: %v", c.input, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

// ─── GoKitHandler ─────────────────────────────────────────────────────────────

// captureLogger is a simple go-kit Logger that records every Log call.
type captureLogger struct {
	mu    sync.Mutex
	calls [][]interface{}
}

func (c *captureLogger) Log(kvs ...interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]interface{}, len(kvs))
	copy(cp, kvs)
	c.calls = append(c.calls, cp)
	return nil
}

// findKV scans a flat key-value slice and returns the value for key.
func findKV(kvs []interface{}, key string) (interface{}, bool) {
	for i := 0; i+1 < len(kvs); i += 2 {
		if k, ok := kvs[i].(string); ok && k == key {
			return kvs[i+1], true
		}
	}
	return nil, false
}

func TestGoKitHandler_BasicRecord(t *testing.T) {
	cap := &captureLogger{}
	logger := slogutil.FromGoKitLogger(cap)

	logger.Info("hello", "user", "alice")

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.calls) != 1 {
		t.Fatalf("expected 1 log call, got %d", len(cap.calls))
	}
	kvs := cap.calls[0]

	if v, ok := findKV(kvs, "msg"); !ok || v != "hello" {
		t.Errorf("expected msg=hello, kvs: %v", kvs)
	}
	if v, ok := findKV(kvs, "user"); !ok || v != "alice" {
		t.Errorf("expected user=alice, kvs: %v", kvs)
	}
}

func TestGoKitHandler_LevelMapping(t *testing.T) {
	cases := []struct {
		emit    func(*slog.Logger)
		wantLvl string
	}{
		{func(l *slog.Logger) { l.Debug("d") }, "debug"},
		{func(l *slog.Logger) { l.Info("i") }, "info"},
		{func(l *slog.Logger) { l.Warn("w") }, "warn"},
		{func(l *slog.Logger) { l.Error("e") }, "error"},
	}
	for _, c := range cases {
		cap := &captureLogger{}
		// AllowAll so go-kit's level filter does not drop debug records.
		filtered := level.NewFilter(cap, level.AllowAll())
		logger := slogutil.FromGoKitLogger(filtered)

		c.emit(logger)

		cap.mu.Lock()
		calls := cap.calls
		cap.mu.Unlock()

		if len(calls) != 1 {
			t.Errorf("level %s: expected 1 call, got %d", c.wantLvl, len(calls))
			continue
		}
		lv, ok := findKV(calls[0], "level")
		if !ok {
			t.Errorf("level %s: no level key in kvs: %v", c.wantLvl, calls[0])
			continue
		}
		lval, ok := lv.(level.Value)
		if !ok {
			t.Errorf("level %s: level value is not level.Value: %T %v", c.wantLvl, lv, lv)
			continue
		}
		if lval.String() != c.wantLvl {
			t.Errorf("expected level=%s, got %s", c.wantLvl, lval.String())
		}
	}
}

func TestGoKitHandler_WithAttrs(t *testing.T) {
	cap := &captureLogger{}
	logger := slogutil.FromGoKitLogger(cap)
	logger = logger.With("component", "store")

	logger.Info("read")

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(cap.calls))
	}
	if v, ok := findKV(cap.calls[0], "component"); !ok || v != "store" {
		t.Errorf("expected component=store in kvs: %v", cap.calls[0])
	}
}

func TestGoKitHandler_WithGroup(t *testing.T) {
	cap := &captureLogger{}
	logger := slogutil.FromGoKitLogger(cap)
	logger = logger.WithGroup("db")
	logger = logger.With("host", "localhost")

	logger.Info("connect")

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(cap.calls))
	}
	// Attrs added after WithGroup should be prefixed with "db.".
	if v, ok := findKV(cap.calls[0], "db.host"); !ok || v != "localhost" {
		t.Errorf("expected db.host=localhost in kvs: %v", cap.calls[0])
	}
}

func TestGoKitHandler_CancelledContextDoesNotError(t *testing.T) {
	cap := &captureLogger{}
	h := slogutil.NewGoKitHandler(cap)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "ctx test", 0)
	if err := h.Handle(ctx, r); err != nil {
		t.Errorf("Handle with cancelled ctx returned error: %v", err)
	}

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.calls) != 1 {
		t.Errorf("expected record to still be emitted, got %d calls", len(cap.calls))
	}
}

// ─── GoKitAdapter ─────────────────────────────────────────────────────────────

func TestGoKitAdapter_BasicLog(t *testing.T) {
	var buf bytes.Buffer
	slogLogger := slogutil.NewLogger(slogutil.Options{
		Writer: &buf,
		Format: slogutil.FormatJSON,
		Level:  slog.LevelDebug,
	})
	adapter := slogutil.NewGoKitAdapter(slogLogger)

	if err := adapter.Log("msg", "something happened", "count", 7); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := mustParseJSON(t, buf.Bytes())
	if m["msg"] != "something happened" {
		t.Errorf("unexpected msg: %v", m["msg"])
	}
	if m["count"] != float64(7) { // JSON numbers decode to float64
		t.Errorf("unexpected count: %v (type %T)", m["count"], m["count"])
	}
}

func TestGoKitAdapter_LevelMapping(t *testing.T) {
	cases := []struct {
		wrapFn  func(kitlog.Logger) kitlog.Logger
		wantLvl string
	}{
		{level.Debug, "DEBUG"},
		{level.Info, "INFO"},
		{level.Warn, "WARN"},
		{level.Error, "ERROR"},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		slogLogger := slogutil.NewLogger(slogutil.Options{
			Writer: &buf,
			Format: slogutil.FormatJSON,
			Level:  slog.LevelDebug,
		})
		adapter := slogutil.NewGoKitAdapter(slogLogger)

		wrapped := c.wrapFn(adapter)
		if err := wrapped.Log("msg", "test"); err != nil {
			t.Errorf("%s: unexpected error: %v", c.wantLvl, err)
			continue
		}

		m := mustParseJSON(t, buf.Bytes())
		if m["level"] != c.wantLvl {
			t.Errorf("expected level=%s, got %v (full: %s)", c.wantLvl, m["level"], buf.String())
		}
	}
}

func TestGoKitAdapter_EmptyLog(t *testing.T) {
	var buf bytes.Buffer
	slogLogger := slogutil.NewLogger(slogutil.Options{
		Writer: &buf,
		Format: slogutil.FormatJSON,
	})
	adapter := slogutil.NewGoKitAdapter(slogLogger)

	if err := adapter.Log(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty Log(), got: %s", buf.String())
	}
}

// ─── round-trip: slog → go-kit → slog ────────────────────────────────────────

func TestRoundTrip_SlogToGokitToSlog(t *testing.T) {
	// Outer slog logger writes JSON to buf.
	var buf bytes.Buffer
	outer := slogutil.NewLogger(slogutil.Options{
		Writer: &buf,
		Format: slogutil.FormatJSON,
		Level:  slog.LevelDebug,
	})

	// Wrap it as a go-kit Logger …
	gokitLogger := slogutil.NewGoKitAdapter(outer)

	// … then wrap it back as a *slog.Logger.
	inner := slogutil.FromGoKitLogger(gokitLogger)

	inner.Error("round-trip test", "attempt", 1)

	m := mustParseJSON(t, buf.Bytes())
	if m["level"] != "ERROR" {
		t.Errorf("expected level=ERROR, got %v", m["level"])
	}
	if m["msg"] != "round-trip test" {
		t.Errorf("unexpected msg: %v", m["msg"])
	}
	if m["attempt"] != float64(1) {
		t.Errorf("unexpected attempt: %v", m["attempt"])
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustParseJSON(t *testing.T, b []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, string(b))
	}
	return m
}
