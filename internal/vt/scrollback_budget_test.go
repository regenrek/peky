package vt

import (
	"testing"
	"unsafe"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestScrollbackPrunesByByteBudget(t *testing.T) {
	cellBytes := int64(unsafe.Sizeof(uv.Cell{}))
	maxBytes := (3 * cellBytes) + 1

	sb := NewScrollback(maxBytes)
	sb.SetWrapWidth(3)

	sb.PushLineWithWrap(mkLine("one", 3), false)
	sb.PushLineWithWrap(mkLine("two", 3), false)

	if got, want := sb.Len(), 1; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}

	row := make([]uv.Cell, 3)
	if !sb.CopyRow(0, row) {
		t.Fatalf("CopyRow(0) failed")
	}
	if got, want := cellsToText(row), "two"; got != want {
		t.Fatalf("row0 = %q, want %q", got, want)
	}
}
