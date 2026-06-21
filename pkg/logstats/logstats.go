// Package logstats provides utilities for collecting, aggregating, and analysing
// statistics about log streams processed by Loki.
//
// The main entry point for most callers is Aggregator, which orchestrates
// per-stream counters, sliding-window rate calculators, and label-cardinality
// analysis into a single consistent snapshot.
package logstats

import (
	"sync"
	"sync/atomic"
	"time"
)

// StreamID is an opaque string that uniquely identifies a log stream.  In
// practice this is the canonical serialised form of a label set, e.g.
// `{app="nginx", env="prod"}`.
type StreamID = string

// Counter holds cumulative byte and line counts for a single stream.
// All fields are updated atomically so the struct can be read and written from
// multiple goroutines without a mutex.
type Counter struct {
	// streamID is the label-set string that identifies this stream.
	streamID StreamID

	// bytes is the total number of bytes ingested for this stream since the
	// Counter was created.
	bytes int64

	// lines is the total number of log lines ingested for this stream since the
	// Counter was created.
	lines int64

	// firstSeen is the time at which the first entry was recorded.
	firstSeen time.Time

	// lastSeen is the time at which the most recent entry was recorded.
	lastSeen time.Time

	mu sync.RWMutex
}

// NewCounter returns a new Counter for the given stream.
func NewCounter(streamID StreamID) *Counter {
	return &Counter{streamID: streamID}
}

// Add records byteCount bytes and lineCount lines for the stream at the current
// wall-clock time.  It is safe to call from multiple goroutines.
func (c *Counter) Add(byteCount, lineCount int64) {
	atomic.AddInt64(&c.bytes, byteCount)
	atomic.AddInt64(&c.lines, lineCount)

	now := time.Now()
	c.mu.Lock()
	if c.firstSeen.IsZero() {
		c.firstSeen = now
	}
	c.lastSeen = now
	c.mu.Unlock()
}

// Bytes returns the cumulative byte count.
func (c *Counter) Bytes() int64 { return atomic.LoadInt64(&c.bytes) }

// Lines returns the cumulative line count.
func (c *Counter) Lines() int64 { return atomic.LoadInt64(&c.lines) }

// StreamID returns the stream identifier associated with this counter.
func (c *Counter) StreamID() StreamID { return c.streamID }

// FirstSeen returns the timestamp of the first recorded entry.
func (c *Counter) FirstSeen() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.firstSeen
}

// LastSeen returns the timestamp of the most recently recorded entry.
func (c *Counter) LastSeen() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastSeen
}

// Reset zeroes all counters and timestamps.  Useful in tests.
func (c *Counter) Reset() {
	atomic.StoreInt64(&c.bytes, 0)
	atomic.StoreInt64(&c.lines, 0)
	c.mu.Lock()
	c.firstSeen = time.Time{}
	c.lastSeen = time.Time{}
	c.mu.Unlock()
}

// Snapshot is an immutable point-in-time copy of a Counter's state.
type Snapshot struct {
	StreamID  StreamID
	Bytes     int64
	Lines     int64
	FirstSeen time.Time
	LastSeen  time.Time
}

// Snapshot returns an immutable copy of the Counter's current state.
func (c *Counter) Snapshot() Snapshot {
	c.mu.RLock()
	fs := c.firstSeen
	ls := c.lastSeen
	c.mu.RUnlock()
	return Snapshot{
		StreamID:  c.streamID,
		Bytes:     atomic.LoadInt64(&c.bytes),
		Lines:     atomic.LoadInt64(&c.lines),
		FirstSeen: fs,
		LastSeen:  ls,
	}
}

// Registry is a thread-safe collection of per-stream Counters.
type Registry struct {
	mu       sync.RWMutex
	counters map[StreamID]*Counter
}

// NewRegistry returns an initialised, empty Registry.
func NewRegistry() *Registry {
	return &Registry{counters: make(map[StreamID]*Counter)}
}

// Record adds byteCount bytes and lineCount log lines to the counter for the
// given stream, creating a new Counter if one does not already exist.
func (r *Registry) Record(id StreamID, byteCount, lineCount int64) {
	r.mu.RLock()
	c, ok := r.counters[id]
	r.mu.RUnlock()
	if ok {
		c.Add(byteCount, lineCount)
		return
	}

	// Counter does not exist – take a write lock and create it.
	r.mu.Lock()
	// Re-check under the write lock to avoid a race between two concurrent
	// Record calls for the same previously-unseen stream.
	if c, ok = r.counters[id]; !ok {
		c = NewCounter(id)
		r.counters[id] = c
	}
	r.mu.Unlock()
	c.Add(byteCount, lineCount)
}

// Get returns the Counter for id, or nil if no entry has been recorded yet.
func (r *Registry) Get(id StreamID) *Counter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.counters[id]
}

// Len returns the number of streams currently tracked.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.counters)
}

// Snapshots returns a copy of all counter snapshots.  The order is not
// guaranteed to be stable across calls.
func (r *Registry) Snapshots() []Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Snapshot, 0, len(r.counters))
	for _, c := range r.counters {
		out = append(out, c.Snapshot())
	}
	return out
}

// Reset removes all counters from the registry.
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters = make(map[StreamID]*Counter)
}
