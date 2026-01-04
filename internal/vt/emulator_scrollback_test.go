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

	_, _ = emu.Write(readTestdataSequence(t, "alt_screen_on.seq"))
	writeLines(emu, 4)
	if emu.ScrollbackLen() != base {
		t.Fatalf("alt screen should not grow scrollback: before=%d after=%d", base, emu.ScrollbackLen())
	}

	_, _ = emu.Write(readTestdataSequence(t, "alt_screen_off.seq"))
	writeLines(emu, 4)
	if emu.ScrollbackLen() <= base {
		t.Fatalf("main screen should grow scrollback after alt: before=%d after=%d", base, emu.ScrollbackLen())
	}
}

func TestResizeReflowsScrollback(t *testing.T) {
	emu := NewEmulator(5, 2)
	sb := emu.Scrollback()
	sb.SetWrapWidth(5)

	sb.PushLineWithWrap(mkLineEmu("hello", 5), true)
	sb.PushLineWithWrap(mkLineEmu(" worl", 5), true)
	sb.PushLineWithWrap(mkLineEmu("d", 5), false)

	emu.Resize(7, 2)

	if sb.Len() != 2 {
		t.Fatalf("expected 2 rows after resize rewrap, got %d", sb.Len())
	}
	row0 := make([]uv.Cell, 7)
	row1 := make([]uv.Cell, 7)
	if !sb.CopyRow(0, row0) || !sb.CopyRow(1, row1) {
		t.Fatalf("CopyRow failed")
	}
	if got := cellsToTextEmu(row0); got != "hello w" {
		t.Fatalf("line0 = %q", got)
	}
	if got := cellsToTextEmu(row1); got != "orld" {
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
