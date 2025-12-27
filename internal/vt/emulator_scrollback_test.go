package vt

import (
	"fmt"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestAltScreenDoesNotAffectScrollback(t *testing.T) {
	emu := NewEmulator(4, 2)

	writeLines(emu, 4)
	base := emu.ScrollbackLen()
	if base == 0 {
		t.Fatalf("expected scrollback to grow on main screen")
	}

	_, _ = emu.Write([]byte("\x1b[?1049h"))
	writeLines(emu, 4)
	if emu.ScrollbackLen() != base {
		t.Fatalf("alt screen should not grow scrollback: before=%d after=%d", base, emu.ScrollbackLen())
	}

	_, _ = emu.Write([]byte("\x1b[?1049l"))
	writeLines(emu, 4)
	if emu.ScrollbackLen() <= base {
		t.Fatalf("main screen should grow scrollback after alt: before=%d after=%d", base, emu.ScrollbackLen())
	}
}

func TestResizeReflowsScrollback(t *testing.T) {
	emu := NewEmulator(5, 2)
	sb := emu.Scrollback()
	sb.SetCaptureWidth(5)

	sb.PushLineWithWrap(mkLineEmu("hello", 5), true)
	sb.PushLineWithWrap(mkLineEmu(" worl", 5), true)
	sb.PushLineWithWrap(mkLineEmu("d", 5), false)

	emu.Resize(7, 2)

	if sb.Len() != 2 {
		t.Fatalf("expected 2 lines after resize reflow, got %d", sb.Len())
	}
	if got := cellsToTextEmu(sb.Line(0)); got != "hello w" {
		t.Fatalf("line0 = %q", got)
	}
	if got := cellsToTextEmu(sb.Line(1)); got != "orld" {
		t.Fatalf("line1 = %q", got)
	}
}

func writeLines(emu *Emulator, count int) {
	var b strings.Builder
	for i := 0; i < count; i++ {
		_, _ = fmt.Fprintf(&b, "L%02d\n", i)
	}
	_, _ = emu.Write([]byte(b.String()))
}

func mkLineEmu(text string, width int) []uv.Cell {
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

func cellsToTextEmu(line []uv.Cell) string {
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
