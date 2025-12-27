package vt

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
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

func wrapFlag(sb *Scrollback, logical int) bool {
	phys := (sb.head + logical) % sb.maxLines
	return sb.softWrapped[phys]
}

func TestScrollbackReflowSoftWrapGroups(t *testing.T) {
	sb := NewScrollback(100)
	sb.SetCaptureWidth(5)

	sb.PushLineWithWrap(mkLine("hello", 5), true)
	sb.PushLineWithWrap(mkLine(" worl", 5), true)
	sb.PushLineWithWrap(mkLine("d", 5), false)

	sb.Reflow(7)

	if sb.Len() != 2 {
		t.Fatalf("expected 2 lines after reflow, got %d", sb.Len())
	}
	if got := cellsToText(sb.Line(0)); got != "hello w" {
		t.Fatalf("line0 = %q", got)
	}
	if got := cellsToText(sb.Line(1)); got != "orld" {
		t.Fatalf("line1 = %q", got)
	}
	if !wrapFlag(sb, 0) {
		t.Fatalf("expected line0 softWrapped=true")
	}
	if wrapFlag(sb, 1) {
		t.Fatalf("expected line1 softWrapped=false")
	}
}

func TestScrollbackReflowPreservesHardBreaks(t *testing.T) {
	sb := NewScrollback(100)
	sb.SetCaptureWidth(4)

	// Two logical lines: "ab" newline "cdefgh"
	sb.PushLineWithWrap(mkLine("ab", 4), false)

	sb.PushLineWithWrap(mkLine("cdef", 4), true)
	sb.PushLineWithWrap(mkLine("gh", 4), false)

	sb.Reflow(6)

	if sb.Len() != 2 {
		t.Fatalf("expected 2 lines after reflow, got %d", sb.Len())
	}
	if got := cellsToText(sb.Line(0)); got != "ab" {
		t.Fatalf("line0 = %q", got)
	}
	if got := cellsToText(sb.Line(1)); got != "cdefgh" {
		t.Fatalf("line1 = %q", got)
	}
}
