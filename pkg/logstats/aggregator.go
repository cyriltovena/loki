package logstats

import (
	"sort"
	"sync"
	"time"
)

// bucket is one time-slice inside a RateCalculator's ring buffer.
type bucket struct {
	ts    time.Time
	bytes int64
	lines int64
}

// RateCalculatorConfig controls the behaviour of a RateCalculator.
type RateCalculatorConfig struct {
	// Window is the total duration over which rates are computed.
	// Buckets outside this window are discarded on each call to Rates().
	Window time.Duration

	// Resolution is the granularity of each bucket.  A smaller value gives
	// more accurate rates but requires more memory.
	Resolution time.Duration
}

// DefaultRateCalculatorConfig returns a sensible default configuration:
// a 1-minute window divided into 6-second buckets (10 buckets total).
func DefaultRateCalculatorConfig() RateCalculatorConfig {
	return RateCalculatorConfig{
		Window:     time.Minute,
		Resolution: 6 * time.Second,
	}
}

// RateCalculator computes sliding-window byte and line rates for a single
// stream using a fixed-size ring buffer of time buckets.
//
// All methods are safe for concurrent use.
type RateCalculator struct {
	cfg     RateCalculatorConfig
	mu      sync.Mutex
	buckets []bucket
	head    int // index of the next bucket to write
	size    int // number of allocated slots
}

// NewRateCalculator returns a RateCalculator configured according to cfg.
// Panics if Window or Resolution is zero, or if Resolution > Window.
func NewRateCalculator(cfg RateCalculatorConfig) *RateCalculator {
	if cfg.Window <= 0 || cfg.Resolution <= 0 {
		panic("logstats: RateCalculator Window and Resolution must be > 0")
	}
	if cfg.Resolution > cfg.Window {
		panic("logstats: RateCalculator Resolution must not exceed Window")
	}
	slots := int(cfg.Window/cfg.Resolution) + 1
	return &RateCalculator{
		cfg:     cfg,
		buckets: make([]bucket, slots),
		size:    slots,
	}
}

// Add records byteCount bytes and lineCount lines at time t.
// Successive calls with the same bucket timestamp are accumulated into the
// same slot; calls with a timestamp in a new bucket advance the ring.
func (rc *RateCalculator) Add(t time.Time, byteCount, lineCount int64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	truncated := t.Truncate(rc.cfg.Resolution)
	current := &rc.buckets[rc.head]

	if current.ts.IsZero() || current.ts.Equal(truncated) {
		// Same bucket or first write – accumulate.
		current.ts = truncated
		current.bytes += byteCount
		current.lines += lineCount
		return
	}

	// Advance to the next slot, overwriting old data.
	rc.head = (rc.head + 1) % rc.size
	rc.buckets[rc.head] = bucket{
		ts:    truncated,
		bytes: byteCount,
		lines: lineCount,
	}
}

// Rates computes the average bytes/s and lines/s over the configured window
// as of time now, discarding buckets older than Window.
func (rc *RateCalculator) Rates(now time.Time) (bytesPerSec, linesPerSec float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	cutoff := now.Add(-rc.cfg.Window)
	var totalBytes, totalLines int64
	var earliest, latest time.Time

	for i := 0; i < rc.size; i++ {
		b := &rc.buckets[i]
		if b.ts.IsZero() || b.ts.Before(cutoff) {
			continue
		}
		totalBytes += b.bytes
		totalLines += b.lines
		if earliest.IsZero() || b.ts.Before(earliest) {
			earliest = b.ts
		}
		if b.ts.After(latest) {
			latest = b.ts
		}
	}

	if earliest.IsZero() {
		return 0, 0
	}

	elapsed := latest.Sub(earliest) + rc.cfg.Resolution
	secs := elapsed.Seconds()
	if secs <= 0 {
		secs = rc.cfg.Resolution.Seconds()
	}
	return float64(totalBytes) / secs, float64(totalLines) / secs
}

// Reset clears all buckets.
func (rc *RateCalculator) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for i := range rc.buckets {
		rc.buckets[i] = bucket{}
	}
	rc.head = 0
}

// ─── Aggregator ──────────────────────────────────────────────────────────────

// AggregatorConfig controls global aggregator behaviour.
type AggregatorConfig struct {
	// TopN is the number of highest-volume streams to include in snapshots.
	TopN int

	// RateConfig is forwarded to each per-stream RateCalculator.
	RateConfig RateCalculatorConfig

	// CardinalityLimit is the maximum number of streams the aggregator will
	// track before it starts logging warnings.  0 means unlimited.
	CardinalityLimit int
}

// DefaultAggregatorConfig returns a ready-to-use configuration.
func DefaultAggregatorConfig() AggregatorConfig {
	return AggregatorConfig{
		TopN:             10,
		RateConfig:       DefaultRateCalculatorConfig(),
		CardinalityLimit: 10_000,
	}
}

// streamEntry holds per-stream state inside the Aggregator.
type streamEntry struct {
	counter *Counter
	rate    *RateCalculator
}

// AggregatorSnapshot is the point-in-time view returned by Aggregator.Snapshot.
type AggregatorSnapshot struct {
	// CapturedAt is the wall-clock time the snapshot was taken.
	CapturedAt time.Time

	// TotalStreams is the number of distinct streams seen since the last Reset.
	TotalStreams int

	// TotalBytes is the sum of all bytes across all streams.
	TotalBytes int64

	// TotalLines is the sum of all log lines across all streams.
	TotalLines int64

	// BytesPerSec is the aggregate byte ingestion rate over the rate window.
	BytesPerSec float64

	// LinesPerSec is the aggregate line ingestion rate over the rate window.
	LinesPerSec float64

	// TopStreams is the TopN streams ordered by descending byte volume.
	TopStreams []StreamStat

	// CardinalityWarning is true when the number of streams exceeds
	// AggregatorConfig.CardinalityLimit.
	CardinalityWarning bool
}

// StreamStat holds per-stream data inside an AggregatorSnapshot.
type StreamStat struct {
	Labels      StreamID
	Bytes       int64
	Lines       int64
	BytesPerSec float64
	LinesPerSec float64
	FirstSeen   time.Time
	LastSeen    time.Time
}

// Aggregator is the main entry point for callers that want a single object
// managing all logstats concerns.  It is safe for concurrent use.
type Aggregator struct {
	cfg AggregatorConfig

	mu      sync.RWMutex
	streams map[StreamID]*streamEntry
}

// NewAggregator returns an Aggregator configured by cfg.
func NewAggregator(cfg AggregatorConfig) *Aggregator {
	return &Aggregator{
		cfg:     cfg,
		streams: make(map[StreamID]*streamEntry),
	}
}

// Record registers byteCount bytes and lineCount log lines for stream id.
func (a *Aggregator) Record(id StreamID, byteCount, lineCount int64) {
	now := time.Now()

	a.mu.RLock()
	entry, ok := a.streams[id]
	a.mu.RUnlock()

	if !ok {
		a.mu.Lock()
		if entry, ok = a.streams[id]; !ok {
			entry = &streamEntry{
				counter: NewCounter(id),
				rate:    NewRateCalculator(a.cfg.RateConfig),
			}
			a.streams[id] = entry
		}
		a.mu.Unlock()
	}

	entry.counter.Add(byteCount, lineCount)
	entry.rate.Add(now, byteCount, lineCount)
}

// Snapshot builds and returns an AggregatorSnapshot.
func (a *Aggregator) Snapshot() AggregatorSnapshot {
	now := time.Now()

	a.mu.RLock()
	entries := make([]*streamEntry, 0, len(a.streams))
	for _, e := range a.streams {
		entries = append(entries, e)
	}
	a.mu.RUnlock()

	snap := AggregatorSnapshot{
		CapturedAt:  now,
		TotalStreams: len(entries),
	}

	stats := make([]StreamStat, 0, len(entries))
	for _, e := range entries {
		cs := e.counter.Snapshot()
		bps, lps := e.rate.Rates(now)
		snap.TotalBytes += cs.Bytes
		snap.TotalLines += cs.Lines
		snap.BytesPerSec += bps
		snap.LinesPerSec += lps
		stats = append(stats, StreamStat{
			Labels:      cs.StreamID,
			Bytes:       cs.Bytes,
			Lines:       cs.Lines,
			BytesPerSec: bps,
			LinesPerSec: lps,
			FirstSeen:   cs.FirstSeen,
			LastSeen:    cs.LastSeen,
		})
	}

	// Sort descending by bytes for the Top-N selection.
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Bytes > stats[j].Bytes
	})

	topN := a.cfg.TopN
	if topN > len(stats) {
		topN = len(stats)
	}
	snap.TopStreams = stats[:topN]

	if a.cfg.CardinalityLimit > 0 && len(entries) > a.cfg.CardinalityLimit {
		snap.CardinalityWarning = true
	}

	return snap
}

// Reset removes all stream state.
func (a *Aggregator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.streams = make(map[StreamID]*streamEntry)
}

// StreamCount returns the number of distinct streams currently tracked.
func (a *Aggregator) StreamCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.streams)
}
