package app

import (
	"regexp"
	"testing"
)

func TestMatchesRunning(t *testing.T) {
	matcher := statusMatcher{running: regexp.MustCompile(`(?m)\brunning\b`)}
	if matchesRunning([]string{"idle"}, matcher) {
		t.Fatalf("expected false")
	}
	if !matchesRunning([]string{"now running"}, matcher) {
		t.Fatalf("expected true")
	}
	if matchesRunning([]string{"now running"}, statusMatcher{}) {
		t.Fatalf("expected false without running matcher")
	}
}
