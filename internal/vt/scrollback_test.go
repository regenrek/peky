package vt

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/regenrek/peakypanes/internal/limits"
)

func mkLine(text string, width int) []uv.Cell {
	runes := []rune(text)
	line := make([]uv.Cell, width)
	for i := 0; i < width; i++ {
		c := uv.EmptyCell
		c.Width = 1
		if i < len(runes) {
			c.Content = string(runes[i])
		}
		line[i] = c
	}
	return line
}

func cellsToText(line []uv.Cell) string {
	var b strings.Builder
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c.Width == 0 {
			continue
		}
		if c.Content == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(c.Content)
		}
	}
	return strings.TrimRight(b.String(), " ")
}

func TestScrollbackReflowSoftWrapGroups(t *testing.T) {
	sb := NewScrollback(limits.TerminalScrollbackMaxBytesDefault)
	sb.SetWrapWidth(5)

	sb.PushLineWithWrap(mkLine("hello", 5), true)
	sb.PushLineWithWrap(mkLine(" worl", 5), true)
	sb.PushLineWithWrap(mkLine("d", 5), false)

	if got, want := sb.Len(), 3; got != want {
		t.Fatalf("Len() at width=5 = %d, want %d", got, want)
	}

	sb.SetWrapWidth(7)

	if sb.Len() != 2 {
		t.Fatalf("expected 2 rows after rewrap, got %d", sb.Len())
	}
	row0 := make([]uv.Cell, 7)
	row1 := make([]uv.Cell, 7)
	if !sb.CopyRow(0, row0) || !sb.CopyRow(1, row1) {
		t.Fatalf("CopyRow failed")
	}
	if got := cellsToText(row0); got != "hello w" {
		t.Fatalf("line0 = %q", got)
	}
	if got := cellsToText(row1); got != "orld" {
		t.Fatalf("line1 = %q", got)
	}
}

func TestScrollbackReflowPreservesHardBreaks(t *testing.T) {
	sb := NewScrollback(limits.TerminalScrollbackMaxBytesDefault)
	sb.SetWrapWidth(4)

	// Two logical lines: "ab" newline "cdefgh"
	sb.PushLineWithWrap(mkLine("ab", 4), false)

	sb.PushLineWithWrap(mkLine("cdef", 4), true)
	sb.PushLineWithWrap(mkLine("gh", 4), false)

	sb.SetWrapWidth(6)

	if sb.Len() != 2 {
		t.Fatalf("expected 2 rows after rewrap, got %d", sb.Len())
	}
	row0 := make([]uv.Cell, 6)
	row1 := make([]uv.Cell, 6)
	if !sb.CopyRow(0, row0) || !sb.CopyRow(1, row1) {
		t.Fatalf("CopyRow failed")
	}
	if got := cellsToText(row0); got != "ab" {
		t.Fatalf("line0 = %q", got)
	}
	if got := cellsToText(row1); got != "cdefgh" {
		t.Fatalf("line1 = %q", got)
	}
}
