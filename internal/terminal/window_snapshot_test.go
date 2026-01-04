package terminal

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestSnapshotPlainCapturesScreenAndScrollback(t *testing.T) {
	emu := &fakeEmu{
		cols: 4,
		rows: 2,
		sb: [][]uv.Cell{
			mkCellsLine("S0", 4),
			mkCellsLine("S1", 4),
		},
		screen: [][]uv.Cell{
			mkCellsLine("A0", 4),
			mkCellsLine("A1", 4),
		},
		cursor: uv.Pos(1, 0),
	}
	w := &Window{term: emu}
	w.altScreen.Store(true)
	w.cursorVisible.Store(true)

	snap, err := w.SnapshotPlain(PlainSnapshotOptions{})
	if err != nil {
		t.Fatalf("SnapshotPlain() error: %v", err)
	}
	if snap.Cols != 4 || snap.Rows != 2 {
		t.Fatalf("size = %dx%d, want 4x2", snap.Cols, snap.Rows)
	}
	if snap.CursorX != 1 || snap.CursorY != 0 {
		t.Fatalf("cursor = %d,%d", snap.CursorX, snap.CursorY)
	}
	if !snap.CursorVisible || !snap.AltScreen {
		t.Fatalf("cursor/alt flags = %v/%v", snap.CursorVisible, snap.AltScreen)
	}
	if len(snap.Scrollback) != 2 || snap.Scrollback[0] != "S0" || snap.Scrollback[1] != "S1" {
		t.Fatalf("scrollback = %#v", snap.Scrollback)
	}
	if len(snap.ScreenLines) != 2 || snap.ScreenLines[0] != "A0" || snap.ScreenLines[1] != "A1" {
		t.Fatalf("screen lines = %#v", snap.ScreenLines)
	}
	if snap.CapturedAt.IsZero() {
		t.Fatalf("expected CapturedAt set")
	}
}

func TestSnapshotPlainMaxScrollbackLines(t *testing.T) {
	emu := &fakeEmu{
		cols: 4,
		rows: 1,
		sb: [][]uv.Cell{
			mkCellsLine("S0", 4),
			mkCellsLine("S1", 4),
			mkCellsLine("S2", 4),
		},
		screen: [][]uv.Cell{
			mkCellsLine("A0", 4),
		},
	}
	w := &Window{term: emu}

	snap, err := w.SnapshotPlain(PlainSnapshotOptions{MaxScrollbackLines: 2})
	if err != nil {
		t.Fatalf("SnapshotPlain() error: %v", err)
	}
	if len(snap.Scrollback) != 2 || snap.Scrollback[0] != "S1" || snap.Scrollback[1] != "S2" {
		t.Fatalf("scrollback = %#v", snap.Scrollback)
	}
}
