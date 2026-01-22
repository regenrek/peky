package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestPekyPolicy_AllowsWithAllowList(t *testing.T) {
	policy := newPekyPolicy(layout.AgentConfig{
		AllowedCommands: []string{"session.*", "pane.send"},
	})

	cases := []struct {
		command string
		want    bool
	}{
		{command: "session.start", want: true},
		{command: "session", want: true},
		{command: "pane.send", want: true},
		{command: "pane.close", want: false},
		{command: "  ", want: false},
	}
	for _, tc := range cases {
		if got := policy.allows(tc.command); got != tc.want {
			t.Fatalf("allows(%q) = %v, want %v", tc.command, got, tc.want)
		}
	}
}

func TestPekyPolicy_AllowsWithBlockList(t *testing.T) {
	policy := newPekyPolicy(layout.AgentConfig{
		BlockedCommands: []string{"pane.*", "unsafe*"},
	})

	cases := []struct {
		command string
		want    bool
	}{
		{command: "pane.send", want: false},
		{command: "unsafeDelete", want: false},
		{command: "session.start", want: true},
	}
	for _, tc := range cases {
		if got := policy.allows(tc.command); got != tc.want {
			t.Fatalf("allows(%q) = %v, want %v", tc.command, got, tc.want)
		}
	}
}

func TestMatchesCommandPattern(t *testing.T) {
	cases := []struct {
		pattern string
		command string
		want    bool
	}{
		{pattern: "pane.send", command: "pane.send", want: true},
		{pattern: "pane.*", command: "pane", want: true},
		{pattern: "pane.*", command: "pane.send", want: true},
		{pattern: "pane.*", command: "panel.send", want: false},
		{pattern: "pane*", command: "pane", want: true},
		{pattern: "pane*", command: "pane.send", want: true},
		{pattern: "session", command: "session", want: true},
		{pattern: "session", command: "session.start", want: true},
		{pattern: "session", command: "sessionx", want: false},
		{pattern: "", command: "session", want: false},
	}
	for _, tc := range cases {
		if got := matchesCommandPattern(tc.pattern, tc.command); got != tc.want {
			t.Fatalf("matchesCommandPattern(%q, %q) = %v, want %v", tc.pattern, tc.command, got, tc.want)
		}
	}
}
