package logsampling

import (
	"testing"
)

func TestShouldKeepAllWhenRateOne(t *testing.T) {
	s := New([]Rule{{LabelMatcher: "app=nginx", Rate: 1}})
	labels := map[string]string{"app": "nginx"}
	for i := 0; i < 100; i++ {
		if !s.ShouldKeep(labels, "log line") {
			t.Fatal("expected all lines to be kept with Rate=1")
		}
	}
}

func TestSamplingReducesLines(t *testing.T) {
	s := New([]Rule{{LabelMatcher: "app=nginx", Rate: 10}})
	labels := map[string]string{"app": "nginx"}
	kept := 0
	total := 1000
	for i := 0; i < total; i++ {
		if s.ShouldKeep(labels, "log line") {
			kept++
		}
	}
	// expect roughly 10% kept, allow wide margin
	if kept < 50 || kept > 200 {
		t.Fatalf("expected ~100 kept out of 1000 with Rate=10, got %d", kept)
	}
}

func TestNoMatchKeepsLine(t *testing.T) {
	s := New([]Rule{{LabelMatcher: "app=mysql", Rate: 100}})
	labels := map[string]string{"app": "nginx"}
	if !s.ShouldKeep(labels, "log line") {
		t.Fatal("non-matching rule should keep the line")
	}
}
