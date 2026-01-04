package vt

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/limits"
)

func TestScrollbackDefaultMaxBytesUsesLimits(t *testing.T) {
	sb := NewScrollback(0)
	if got, want := sb.MaxBytes(), limits.TerminalScrollbackMaxBytesDefault; got != want {
		t.Fatalf("MaxBytes() = %d, want %d", got, want)
	}
}

func TestScrollbackMaxBytesClamped(t *testing.T) {
	sb := NewScrollback(limits.TerminalScrollbackMaxBytesMax + 123)
	if got, want := sb.MaxBytes(), limits.TerminalScrollbackMaxBytesMax; got != want {
		t.Fatalf("MaxBytes() = %d, want %d", got, want)
	}
}
