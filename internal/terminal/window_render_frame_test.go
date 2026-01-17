package terminal

import (
	"context"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/regenrek/peakypanes/internal/termframe"
)

func TestViewFrameIncludesStyleAndCursor(t *testing.T) {
	row := []uv.Cell{
		{Content: "A", Width: 1},
		{Content: "B", Width: 1, Style: uv.Style{Fg: ansi.BasicColor(2)}},
		{Content: "C", Width: 1},
	}
	emu := &fakeEmu{
		cols:   3,
		rows:   1,
		screen: [][]uv.Cell{row},
		cursor: uv.Pos(2, 0),
	}
	w := &Window{
		term:    emu,
		cols:    3,
		rows:    1,
		updates: make(chan struct{}, 1),
	}
	w.cursorVisible.Store(true)

	frame, err := w.ViewFrameCtx(context.Background())
	if err != nil {
		t.Fatalf("ViewFrameCtx error: %v", err)
	}
	if !frame.Cursor.Visible || frame.Cursor.X != 2 || frame.Cursor.Y != 0 {
		t.Fatalf("unexpected cursor state: %#v", frame.Cursor)
	}
	cell := frame.CellAt(1, 0)
	if cell == nil {
		t.Fatalf("expected cell at 1,0")
	}
	if cell.Style.Fg.Kind != termframe.ColorBasic || cell.Style.Fg.Value != 2 {
		t.Fatalf("unexpected style: %#v", cell.Style)
	}
}

func TestViewFrameSelectionHighlight(t *testing.T) {
	row := []uv.Cell{
		{Content: "a", Width: 1},
		{Content: "b", Width: 1},
		{Content: "c", Width: 1},
		{Content: "d", Width: 1},
	}
	emu := &fakeEmu{
		cols:   4,
		rows:   1,
		screen: [][]uv.Cell{row},
		cursor: uv.Pos(1, 0),
	}
	w := &Window{
		term:    emu,
		cols:    4,
		rows:    1,
		updates: make(chan struct{}, 1),
	}
	w.CopyMode = &CopyMode{
		Active:       true,
		Selecting:    true,
		CursorX:      1,
		CursorAbsY:   0,
		SelStartX:    1,
		SelStartAbsY: 0,
		SelEndX:      2,
		SelEndAbsY:   0,
	}

	frame, err := w.ViewFrameCtx(context.Background())
	if err != nil {
		t.Fatalf("ViewFrameCtx error: %v", err)
	}
	selected := frame.CellAt(2, 0)
	if selected == nil || selected.Style.Attrs&termframe.AttrReverse == 0 {
		t.Fatalf("expected selected cell reverse")
	}
	cursor := frame.CellAt(1, 0)
	if cursor == nil || cursor.Style.Attrs&termframe.AttrBold == 0 {
		t.Fatalf("expected cursor cell bold")
	}
}

func TestViewFrameHidesCursorInScrollback(t *testing.T) {
	emu := &fakeEmu{
		cols:   2,
		rows:   1,
		sb:     [][]uv.Cell{mkCellsLine("S0", 2)},
		screen: [][]uv.Cell{mkCellsLine("A0", 2)},
		cursor: uv.Pos(0, 0),
	}
	w := &Window{
		term:    emu,
		cols:    2,
		rows:    1,
		updates: make(chan struct{}, 1),
	}
	w.cursorVisible.Store(true)
	w.ScrollbackMode = true
	w.ScrollbackOffset = 1

	frame, err := w.ViewFrameCtx(context.Background())
	if err != nil {
		t.Fatalf("ViewFrameCtx error: %v", err)
	}
	if frame.Cursor.Visible {
		t.Fatalf("expected cursor hidden in scrollback")
	}
}

func TestRefreshFrameCacheSignalsWhenSettled(t *testing.T) {
	emu := &fakeEmu{
		cols:   2,
		rows:   1,
		screen: [][]uv.Cell{mkCellsLine("OK", 2)},
	}
	w := &Window{
		term:    emu,
		cols:    2,
		rows:    1,
		updates: make(chan struct{}, 1),
	}
	w.TouchFrameDemand(0)

	w.cacheMu.Lock()
	w.cacheDirty = true
	w.cacheMu.Unlock()
	w.updateSeq.Store(10)

	w.refreshFrameCache()

	select {
	case <-w.updates:
	default:
		t.Fatalf("expected updates signal when cache settles")
	}
}

func TestRefreshFrameCacheDoesNotSignalWhenAlreadyClean(t *testing.T) {
	emu := &fakeEmu{
		cols:   2,
		rows:   1,
		screen: [][]uv.Cell{mkCellsLine("OK", 2)},
	}
	w := &Window{
		term:    emu,
		cols:    2,
		rows:    1,
		updates: make(chan struct{}, 1),
	}
	w.TouchFrameDemand(0)
	w.updateSeq.Store(10)

	w.refreshFrameCache()

	select {
	case <-w.updates:
		t.Fatalf("unexpected updates signal for clean cache")
	default:
	}
}
