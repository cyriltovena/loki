package logsampling

import (
	"math/rand"
	"sync"
)

// Rule defines a sampling rule: lines matching the label selector are sampled at Rate (1/N).
type Rule struct {
	LabelMatcher string // e.g. `{app="nginx"}`
	Rate         int    // keep 1 in Rate lines; 1 = keep all, 10 = keep 10%
}

// Sampler applies rate-based sampling rules to log lines.
type Sampler struct {
	mu    sync.Mutex
	rules []Rule
	rng   *rand.Rand
}

// New creates a Sampler with the given rules.
func New(rules []Rule) *Sampler {
	return &Sampler{
		rules: rules,
		rng:   rand.New(rand.NewSource(42)),
	}
}

// ShouldKeep returns true if the log line should be kept based on the matching rule.
// It matches rules in order and applies the first match. If no rule matches, the line is kept.
func (s *Sampler) ShouldKeep(labels map[string]string, _ string) bool {
	for _, rule := range s.rules {
		if matchesSelector(labels, rule.LabelMatcher) {
			if rule.Rate <= 1 {
				return true
			}
			s.mu.Lock()
			n := s.rng.Intn(rule.Rate)
			s.mu.Unlock()
			return n == 0
		}
	}
	return true
}

// matchesSelector is a simplified label matcher (exact key=value pairs only).
func matchesSelector(labels map[string]string, selector string) bool {
	if selector == "" {
		return true
	}
	// naive: check if all key=value pairs in selector exist in labels
	// Format expected: key=value (comma-separated, no braces for simplicity)
	import_parts := splitSelector(selector)
	for _, part := range import_parts {
		kv := splitKV(part)
		if len(kv) != 2 {
			continue
		}
		if v, ok := labels[kv[0]]; !ok || v != kv[1] {
			return false
		}
	}
	return true
}

func splitSelector(s string) []string {
	var parts []string
	cur := ""
	for _, c := range s {
		if c == ',' {
			parts = append(parts, cur)
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

func splitKV(s string) []string {
	for i, c := range s {
		if c == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
