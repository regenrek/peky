package terminal

import (
	"os"
	"os/exec"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestClampViewStateClampsCopyCursor(t *testing.T) {
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
	}
	w := &Window{
		term:    emu,
		cols:    4,
		rows:    2,
		updates: make(chan struct{}, 1),
	}
	w.ScrollbackOffset = 10
	w.CopyMode = &CopyMode{Active: true, CursorX: -5, CursorAbsY: 99}
	w.clampViewState()
	if w.ScrollbackOffset < 0 || w.ScrollbackOffset > len(emu.sb) {
		t.Fatalf("expected scrollback offset in range, got %d", w.ScrollbackOffset)
	}
	if w.CopyMode.CursorX != 0 {
		t.Fatalf("expected cursor x clamped, got %d", w.CopyMode.CursorX)
	}
	total := len(emu.sb) + w.rows
	if w.CopyMode.CursorAbsY != total-1 {
		t.Fatalf("expected cursor y clamped, got %d", w.CopyMode.CursorAbsY)
	}

	w.CopyMode = nil
	w.ScrollbackMode = true
	w.ScrollbackOffset = 0
	w.clampViewState()
	if w.ScrollbackMode {
		t.Fatalf("expected scrollback mode disabled at offset 0")
	}
}

func TestCopyToggleSelectToggles(t *testing.T) {
	w := &Window{
		CopyMode: &CopyMode{Active: true, CursorX: 1, CursorAbsY: 2},
		updates:  make(chan struct{}, 1),
	}
	w.CopyToggleSelect()
	if !w.CopyMode.Selecting {
		t.Fatalf("expected selection enabled")
	}
	if w.CopyMode.SelStartX != 1 || w.CopyMode.SelStartAbsY != 2 {
		t.Fatalf("unexpected selection start")
	}

	w.CopyToggleSelect()
	if w.CopyMode.Selecting {
		t.Fatalf("expected selection disabled")
	}
}

func TestAdjustToNonContinuation(t *testing.T) {
	line := []uv.Cell{
		{Content: "A", Width: 1},
		{Content: "", Width: 0},
		{Content: "B", Width: 1},
	}
	emu := &fakeEmu{
		cols:   3,
		rows:   1,
		screen: [][]uv.Cell{line},
	}
	w := &Window{
		term: emu,
		cols: 3,
		rows: 1,
	}
	got := w.adjustToNonContinuation(0, 1, 1)
	if got != 2 {
		t.Fatalf("expected adjustment to 2, got %d", got)
	}
}

func TestWindowPID(t *testing.T) {
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	w := &Window{cmd: &exec.Cmd{Process: proc}}
	if w.PID() != os.Getpid() {
		t.Fatalf("PID = %d", w.PID())
	}
	var nilWin *Window
	if nilWin.PID() != 0 {
		t.Fatalf("expected nil PID 0")
	}
}
