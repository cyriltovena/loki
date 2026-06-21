package logstats

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ─── Counter tests ────────────────────────────────────────────────────────────

func TestCounter_BasicAddAndRead(t *testing.T) {
	c := NewCounter("app=test")
	c.Add(100, 5)

	if got := c.Bytes(); got != 100 {
		t.Errorf("Bytes() = %d, want 100", got)
	}
	if got := c.Lines(); got != 5 {
		t.Errorf("Lines() = %d, want 5", got)
	}
	if c.StreamID() != "app=test" {
		t.Errorf("StreamID() = %q, want %q", c.StreamID(), "app=test")
	}
}

func TestCounter_MultipleAdds(t *testing.T) {
	c := NewCounter("svc=db")
	for i := 0; i < 10; i++ {
		c.Add(50, 1)
	}
	if got := c.Bytes(); got != 500 {
		t.Errorf("Bytes() = %d, want 500", got)
	}
	if got := c.Lines(); got != 10 {
		t.Errorf("Lines() = %d, want 10", got)
	}
}

func TestCounter_Reset(t *testing.T) {
	c := NewCounter("svc=cache")
	c.Add(1000, 20)
	c.Reset()

	if c.Bytes() != 0 || c.Lines() != 0 {
		t.Errorf("after Reset(): Bytes=%d Lines=%d, want both 0", c.Bytes(), c.Lines())
	}
	if !c.FirstSeen().IsZero() || !c.LastSeen().IsZero() {
		t.Error("timestamps should be zero after Reset()")
	}
}

func TestCounter_ConcurrentAdd(t *testing.T) {
	c := NewCounter("concurrent")
	var wg sync.WaitGroup
	const goroutines = 50
	const addsPerGoroutine = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < addsPerGoroutine; j++ {
				c.Add(1, 1)
			}
		}()
	}
	wg.Wait()

	want := int64(goroutines * addsPerGoroutine)
	if c.Bytes() != want {
		t.Errorf("Bytes() = %d, want %d", c.Bytes(), want)
	}
	if c.Lines() != want {
		t.Errorf("Lines() = %d, want %d", c.Lines(), want)
	}
}

func TestCounter_Snapshot(t *testing.T) {
	c := NewCounter("snap=test")
	c.Add(256, 4)
	s := c.Snapshot()

	if s.StreamID != "snap=test" {
		t.Errorf("Snapshot StreamID = %q", s.StreamID)
	}
	if s.Bytes != 256 {
		t.Errorf("Snapshot Bytes = %d, want 256", s.Bytes)
	}
	if s.Lines != 4 {
		t.Errorf("Snapshot Lines = %d, want 4", s.Lines)
	}
	if s.FirstSeen.IsZero() {
		t.Error("Snapshot FirstSeen should not be zero")
	}
}

// ─── Registry tests ───────────────────────────────────────────────────────────

func TestRegistry_RecordAndGet(t *testing.T) {
	r := NewRegistry()
	r.Record("svc=api", 100, 1)
	r.Record("svc=api", 200, 2)

	c := r.Get("svc=api")
	if c == nil {
		t.Fatal("expected counter, got nil")
	}
	if c.Bytes() != 300 {
		t.Errorf("Bytes() = %d, want 300", c.Bytes())
	}
}

func TestRegistry_MultipleStreams(t *testing.T) {
	r := NewRegistry()
	streams := []string{"a=1", "b=2", "c=3"}
	for _, s := range streams {
		r.Record(s, 10, 1)
	}
	if r.Len() != 3 {
		t.Errorf("Len() = %d, want 3", r.Len())
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	r := NewRegistry()
	if r.Get("nonexistent") != nil {
		t.Error("expected nil for unknown stream")
	}
}

func TestRegistry_Snapshots(t *testing.T) {
	r := NewRegistry()
	r.Record("x=1", 1, 1)
	r.Record("x=2", 2, 2)
	snaps := r.Snapshots()
	if len(snaps) != 2 {
		t.Errorf("Snapshots() returned %d items, want 2", len(snaps))
	}
}

func TestRegistry_Reset(t *testing.T) {
	r := NewRegistry()
	r.Record("z=1", 1, 1)
	r.Reset()
	if r.Len() != 0 {
		t.Errorf("after Reset() Len() = %d, want 0", r.Len())
	}
}

func TestRegistry_ConcurrentRecord(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	const goroutines = 20
	const streams = 5

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("stream=%d", idx%streams)
			r.Record(id, 1, 1)
		}(i)
	}
	wg.Wait()

	if r.Len() != streams {
		t.Errorf("Len() = %d, want %d", r.Len(), streams)
	}
}

// ─── RateCalculator tests ─────────────────────────────────────────────────────

func TestRateCalculator_ZeroWithoutData(t *testing.T) {
	rc := NewRateCalculator(DefaultRateCalculatorConfig())
	bps, lps := rc.Rates(time.Now())
	if bps != 0 || lps != 0 {
		t.Errorf("expected zero rates, got bps=%f lps=%f", bps, lps)
	}
}

func TestRateCalculator_BasicRate(t *testing.T) {
	cfg := RateCalculatorConfig{Window: 10 * time.Second, Resolution: time.Second}
	rc := NewRateCalculator(cfg)

	base := time.Now().Truncate(time.Second)
	// Add data spread across 5 distinct seconds.
	for i := 0; i < 5; i++ {
		rc.Add(base.Add(time.Duration(i)*time.Second), 100, 1)
	}

	// Compute rates relative to just after the last bucket.
	bps, lps := rc.Rates(base.Add(5 * time.Second))
	if bps <= 0 {
		t.Errorf("expected positive bps, got %f", bps)
	}
	if lps <= 0 {
		t.Errorf("expected positive lps, got %f", lps)
	}
}

func TestRateCalculator_OldDataExcluded(t *testing.T) {
	cfg := RateCalculatorConfig{Window: 5 * time.Second, Resolution: time.Second}
	rc := NewRateCalculator(cfg)

	old := time.Now().Add(-10 * time.Second)
	rc.Add(old, 9999, 9999)

	bps, lps := rc.Rates(time.Now())
	if bps != 0 || lps != 0 {
		t.Errorf("expected zero after window expiry, got bps=%f lps=%f", bps, lps)
	}
}

func TestRateCalculator_Reset(t *testing.T) {
	rc := NewRateCalculator(DefaultRateCalculatorConfig())
	rc.Add(time.Now(), 100, 1)
	rc.Reset()
	bps, lps := rc.Rates(time.Now())
	if bps != 0 || lps != 0 {
		t.Errorf("expected zero after Reset, got bps=%f lps=%f", bps, lps)
	}
}

func TestNewRateCalculator_PanicsOnBadConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for zero Window")
		}
	}()
	NewRateCalculator(RateCalculatorConfig{Window: 0, Resolution: time.Second})
}

// ─── Aggregator tests ─────────────────────────────────────────────────────────

func TestAggregator_RecordAndSnapshot(t *testing.T) {
	a := NewAggregator(DefaultAggregatorConfig())
	a.Record("app=nginx,env=prod", 1024, 10)
	a.Record("app=mysql,env=prod", 512, 5)
	a.Record("app=nginx,env=prod", 2048, 20)

	snap := a.Snapshot()

	if snap.TotalStreams != 2 {
		t.Errorf("TotalStreams = %d, want 2", snap.TotalStreams)
	}
	if snap.TotalBytes != 3584 {
		t.Errorf("TotalBytes = %d, want 3584", snap.TotalBytes)
	}
	if snap.TotalLines != 35 {
		t.Errorf("TotalLines = %d, want 35", snap.TotalLines)
	}
}

func TestAggregator_TopNOrdering(t *testing.T) {
	cfg := DefaultAggregatorConfig()
	cfg.TopN = 2
	a := NewAggregator(cfg)

	a.Record("stream=low", 10, 1)
	a.Record("stream=high", 9000, 9)
	a.Record("stream=mid", 500, 5)

	snap := a.Snapshot()
	if len(snap.TopStreams) != 2 {
		t.Fatalf("TopStreams len = %d, want 2", len(snap.TopStreams))
	}
	if snap.TopStreams[0].Labels != "stream=high" {
		t.Errorf("TopStreams[0] = %q, want stream=high", snap.TopStreams[0].Labels)
	}
	if snap.TopStreams[1].Labels != "stream=mid" {
		t.Errorf("TopStreams[1] = %q, want stream=mid", snap.TopStreams[1].Labels)
	}
}

func TestAggregator_CardinalityWarning(t *testing.T) {
	cfg := DefaultAggregatorConfig()
	cfg.CardinalityLimit = 2
	a := NewAggregator(cfg)

	a.Record("s=1", 1, 1)
	a.Record("s=2", 1, 1)
	a.Record("s=3", 1, 1) // exceeds limit

	snap := a.Snapshot()
	if !snap.CardinalityWarning {
		t.Error("expected CardinalityWarning=true when stream count exceeds limit")
	}
}

func TestAggregator_NoCardinalityWarningUnderLimit(t *testing.T) {
	cfg := DefaultAggregatorConfig()
	cfg.CardinalityLimit = 100
	a := NewAggregator(cfg)

	a.Record("only=one", 1, 1)

	snap := a.Snapshot()
	if snap.CardinalityWarning {
		t.Error("unexpected CardinalityWarning when under limit")
	}
}

func TestAggregator_Reset(t *testing.T) {
	a := NewAggregator(DefaultAggregatorConfig())
	a.Record("app=x", 100, 1)
	a.Reset()

	if a.StreamCount() != 0 {
		t.Errorf("StreamCount after Reset = %d, want 0", a.StreamCount())
	}
}

func TestAggregator_ConcurrentRecord(t *testing.T) {
	a := NewAggregator(DefaultAggregatorConfig())
	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("worker=%d", n%10)
			a.Record(id, 128, 1)
		}(i)
	}
	wg.Wait()

	snap := a.Snapshot()
	if snap.TotalStreams != 10 {
		t.Errorf("TotalStreams = %d, want 10", snap.TotalStreams)
	}
	if snap.TotalBytes != int64(goroutines*128) {
		t.Errorf("TotalBytes = %d, want %d", snap.TotalBytes, goroutines*128)
	}
}

// ─── Labels tests ─────────────────────────────────────────────────────────────

func TestParseLabels_Basic(t *testing.T) {
	pairs := ParseLabels("app=nginx,env=prod")
	if len(pairs) != 2 {
		t.Fatalf("ParseLabels returned %d pairs, want 2", len(pairs))
	}
	if pairs[0].Name != "app" || pairs[0].Value != "nginx" {
		t.Errorf("pairs[0] = %+v", pairs[0])
	}
	if pairs[1].Name != "env" || pairs[1].Value != "prod" {
		t.Errorf("pairs[1] = %+v", pairs[1])
	}
}

func TestParseLabels_Empty(t *testing.T) {
	if pairs := ParseLabels(""); len(pairs) != 0 {
		t.Errorf("ParseLabels(\"\") returned %d pairs, want 0", len(pairs))
	}
}

func TestParseLabels_Whitespace(t *testing.T) {
	pairs := ParseLabels("  app = nginx , env = prod  ")
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	if pairs[0].Name != "app" || pairs[0].Value != "nginx" {
		t.Errorf("unexpected pair: %+v", pairs[0])
	}
}

func TestNormaliseLabels_Sorting(t *testing.T) {
	pairs := []LabelPair{
		{Name: "z", Value: "last"},
		{Name: "a", Value: "first"},
		{Name: "m", Value: "middle"},
	}
	norm := NormaliseLabels(pairs)
	if norm[0].Name != "a" || norm[1].Name != "m" || norm[2].Name != "z" {
		t.Errorf("unexpected order: %+v", norm)
	}
}

func TestNormaliseLabels_Deduplication(t *testing.T) {
	pairs := []LabelPair{
		{Name: "k", Value: "first"},
		{Name: "k", Value: "second"},
	}
	norm := NormaliseLabels(pairs)
	if len(norm) != 1 {
		t.Fatalf("expected 1 pair after dedup, got %d", len(norm))
	}
	// Last value wins.
	if norm[0].Value != "second" {
		t.Errorf("expected value=second, got %q", norm[0].Value)
	}
}

func TestLabelsToString(t *testing.T) {
	pairs := []LabelPair{
		{Name: "env", Value: "prod"},
		{Name: "app", Value: "nginx"},
	}
	got := LabelsToString(pairs)
	// Should be sorted alphabetically.
	want := "app=nginx,env=prod"
	if got != want {
		t.Errorf("LabelsToString = %q, want %q", got, want)
	}
}

func TestCardinalityTracker_BasicObserve(t *testing.T) {
	ct := NewCardinalityTracker()
	ct.Observe([]LabelPair{{Name: "env", Value: "prod"}})
	ct.Observe([]LabelPair{{Name: "env", Value: "staging"}})
	ct.Observe([]LabelPair{{Name: "env", Value: "prod"}}) // duplicate

	if got := ct.Cardinality("env"); got != 2 {
		t.Errorf("Cardinality(env) = %d, want 2", got)
	}
}

func TestCardinalityTracker_HighCardinality(t *testing.T) {
	ct := NewCardinalityTracker()
	for i := 0; i < 20; i++ {
		ct.Observe([]LabelPair{{Name: "trace_id", Value: fmt.Sprintf("t%d", i)}})
	}
	ct.Observe([]LabelPair{{Name: "env", Value: "prod"}})

	high := ct.HighCardinalityLabels(5)
	if len(high) != 1 {
		t.Fatalf("expected 1 high-cardinality label, got %d", len(high))
	}
	if high[0].Name != "trace_id" {
		t.Errorf("expected trace_id to be high-cardinality, got %q", high[0].Name)
	}
	if high[0].Count != 20 {
		t.Errorf("expected count=20, got %d", high[0].Count)
	}
}

func TestCardinalityTracker_LabelNames(t *testing.T) {
	ct := NewCardinalityTracker()
	ct.Observe([]LabelPair{{Name: "z", Value: "v"}})
	ct.Observe([]LabelPair{{Name: "a", Value: "v"}})

	names := ct.LabelNames()
	if len(names) != 2 || names[0] != "a" || names[1] != "z" {
		t.Errorf("LabelNames() = %v", names)
	}
}

func TestCardinalityTracker_AllCardinalities(t *testing.T) {
	ct := NewCardinalityTracker()
	ct.Observe([]LabelPair{
		{Name: "app", Value: "a"},
		{Name: "app", Value: "b"},
		{Name: "env", Value: "prod"},
	})
	m := ct.AllCardinalities()
	if m["app"] != 2 {
		t.Errorf("app cardinality = %d, want 2", m["app"])
	}
	if m["env"] != 1 {
		t.Errorf("env cardinality = %d, want 1", m["env"])
	}
}

func TestCardinalityTracker_Reset(t *testing.T) {
	ct := NewCardinalityTracker()
	ct.Observe([]LabelPair{{Name: "k", Value: "v"}})
	ct.Reset()
	if ct.Cardinality("k") != 0 {
		t.Error("expected 0 after Reset()")
	}
}

func TestCardinalityTracker_ConcurrentObserve(t *testing.T) {
	ct := NewCardinalityTracker()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ct.Observe([]LabelPair{{Name: "worker", Value: fmt.Sprintf("%d", n%5)}})
		}(i)
	}
	wg.Wait()
	if got := ct.Cardinality("worker"); got != 5 {
		t.Errorf("Cardinality(worker) = %d, want 5", got)
	}
}
