package logstats

import (
	"sort"
	"strings"
	"sync"
)

// LabelPair is a single key=value label.
type LabelPair struct {
	Name  string
	Value string
}

// ParseLabels parses a simple comma-separated label string of the form
// `key=value,key=value,...` and returns the constituent pairs in the order
// they appear.  Whitespace around keys and values is trimmed.
//
// This is intentionally simple and does not handle quoted values or escaped
// characters; use the full label-parsing path for production label sets.
func ParseLabels(s string) []LabelPair {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]LabelPair, 0, len(parts))
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		out = append(out, LabelPair{
			Name:  strings.TrimSpace(kv[0]),
			Value: strings.TrimSpace(kv[1]),
		})
	}
	return out
}

// NormaliseLabels sorts labels by name and deduplicates, keeping the last
// value seen for each key.  The resulting slice is in alphabetical order,
// which gives a canonical representation for a label set.
func NormaliseLabels(pairs []LabelPair) []LabelPair {
	if len(pairs) == 0 {
		return nil
	}
	// Deduplicate: last write wins.
	seen := make(map[string]string, len(pairs))
	for _, p := range pairs {
		seen[p.Name] = p.Value
	}
	out := make([]LabelPair, 0, len(seen))
	for k, v := range seen {
		out = append(out, LabelPair{Name: k, Value: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// LabelsToString serialises a slice of LabelPairs into the canonical
// `key=value,key=value` form.  Pairs are sorted by name before serialisation
// so that the output is stable regardless of input order.
func LabelsToString(pairs []LabelPair) string {
	normalised := NormaliseLabels(pairs)
	sb := strings.Builder{}
	for i, p := range normalised {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(p.Name)
		sb.WriteByte('=')
		sb.WriteString(p.Value)
	}
	return sb.String()
}

// CardinalityTracker counts distinct values per label name across all streams
// recorded into it.  It can be used to detect high-cardinality labels before
// they reach the index.
//
// CardinalityTracker is safe for concurrent use.
type CardinalityTracker struct {
	mu     sync.RWMutex
	values map[string]map[string]struct{} // name → set of distinct values
}

// NewCardinalityTracker returns an empty CardinalityTracker.
func NewCardinalityTracker() *CardinalityTracker {
	return &CardinalityTracker{values: make(map[string]map[string]struct{})}
}

// Observe records the label pairs in pairs into the tracker.
func (ct *CardinalityTracker) Observe(pairs []LabelPair) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	for _, p := range pairs {
		if _, ok := ct.values[p.Name]; !ok {
			ct.values[p.Name] = make(map[string]struct{})
		}
		ct.values[p.Name][p.Value] = struct{}{}
	}
}

// Cardinality returns the number of distinct values seen for the given label
// name.  Returns 0 if the label has never been observed.
func (ct *CardinalityTracker) Cardinality(labelName string) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return len(ct.values[labelName])
}

// LabelNames returns a sorted list of all label names observed so far.
func (ct *CardinalityTracker) LabelNames() []string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	names := make([]string, 0, len(ct.values))
	for n := range ct.values {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// HighCardinalityLabels returns label names whose distinct-value count exceeds
// threshold, ordered by cardinality (highest first).
func (ct *CardinalityTracker) HighCardinalityLabels(threshold int) []CardinalityEntry {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var out []CardinalityEntry
	for name, vals := range ct.values {
		if len(vals) > threshold {
			out = append(out, CardinalityEntry{Name: name, Count: len(vals)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

// CardinalityEntry is returned by HighCardinalityLabels.
type CardinalityEntry struct {
	Name  string
	Count int
}

// Reset clears all observed data.
func (ct *CardinalityTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.values = make(map[string]map[string]struct{})
}

// AllCardinalities returns a map of label name to distinct value count for
// every label observed so far.
func (ct *CardinalityTracker) AllCardinalities() map[string]int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	out := make(map[string]int, len(ct.values))
	for name, vals := range ct.values {
		out[name] = len(vals)
	}
	return out
}
