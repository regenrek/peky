package vt

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/limits"
)

func TestScrollbackDefaultMaxLinesUsesLimits(t *testing.T) {
	sb := NewScrollback(0)
	if got, want := sb.MaxLines(), limits.TerminalScrollbackMaxLinesDefault; got != want {
		t.Fatalf("MaxLines() = %d, want %d", got, want)
	}
}

func TestScrollbackMaxLinesClamped(t *testing.T) {
	sb := NewScrollback(limits.TerminalScrollbackMaxLinesMax + 123)
	if got, want := sb.MaxLines(), limits.TerminalScrollbackMaxLinesMax; got != want {
		t.Fatalf("MaxLines() = %d, want %d", got, want)
	}
}
