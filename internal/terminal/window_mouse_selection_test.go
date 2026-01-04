package terminal

import (
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func TestMouseDragSelectionEntersCopyMode(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 3,
		sb: [][]uv.Cell{
			mkCellsLine("S0", 5),
			mkCellsLine("S1", 5),
			mkCellsLine("S2", 5),
			mkCellsLine("S3", 5),
			mkCellsLine("S4", 5),
		},
		screen: [][]uv.Cell{
			mkCellsLine("A0", 5),
			mkCellsLine("A1", 5),
			mkCellsLine("A2", 5),
		},
		cursor: uv.Pos(0, 0),
	}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    3,
		updates: make(chan struct{}, 10),
	}

	assertCopyModeInactive(t, w)

	emu.sentMouse = nil
	assertSendMouseHandled(t, w, uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto, "expected click handled for selection candidate")
	assertCopyModeInactive(t, w)
	assertMouseNotForwarded(t, emu)

	emu.sentMouse = nil
	assertSendMouseHandled(t, w, uv.MouseMotionEvent{X: 3, Y: 2, Button: uv.MouseNone}, MouseRouteAuto, "expected drag motion to start selection")
	assertCopyModeActive(t, w)
	assertCopySelecting(t, w)
	assertMouseSelectionActive(t, w)
	// y=2 => absY=7
	assertSelectionCursor(t, w, 3, 7)
	assertSelectionStartEnd(t, w, 1, 5, 3, 7)
	assertMouseNotForwarded(t, emu)

	emu.sentMouse = nil
	assertSendMouseHandled(t, w, uv.MouseReleaseEvent{X: 3, Y: 2, Button: uv.MouseLeft}, MouseRouteAuto, "expected release handled for selection")
	assertMouseSelectionActive(t, w)
	assertCopyModeActive(t, w)
	assertCopySelecting(t, w)
	assertMouseNotForwarded(t, emu)
}

func TestMouseSelectionAutoCopiesOnRelease(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 1,
		screen: [][]uv.Cell{
			mkCellsLine("hello", 5),
		},
		cursor: uv.Pos(0, 0),
	}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    1,
		updates: make(chan struct{}, 2),
	}

	orig := writeClipboard
	defer func() { writeClipboard = orig }()
	var copied string
	var toast string
	writeClipboard = func(text string) error {
		copied = text
		return nil
	}
	w.toastFn = func(message string) {
		toast = message
	}

	if !w.SendMouse(uv.MouseClickEvent{X: 0, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected click handled for selection candidate")
	}
	if !w.SendMouse(uv.MouseMotionEvent{X: 4, Y: 0, Button: uv.MouseNone}, MouseRouteAuto) {
		t.Fatalf("expected motion to extend selection")
	}
	if !w.SendMouse(uv.MouseReleaseEvent{X: 4, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected release to finish selection")
	}
	if copied == "" {
		t.Fatalf("expected copied text, got empty")
	}
	if !strings.Contains(copied, "hello") {
		t.Fatalf("expected copied text to include hello, got %q", copied)
	}
	if toast != mouseSelectionCopiedToast {
		t.Fatalf("expected toast %q, got %q", mouseSelectionCopiedToast, toast)
	}
}

func TestMouseSelectionDoesNotStealMouseWhenMouseModeEnabled(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 3,
		screen: [][]uv.Cell{
			mkCellsLine("A0", 5),
			mkCellsLine("A1", 5),
			mkCellsLine("A2", 5),
		},
	}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    3,
		updates: make(chan struct{}, 2),
	}
	w.updateMouseMode(ansi.ModeMouseX10, true)

	emu.sentMouse = nil
	if !w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected click forwarded when mouse mode enabled")
	}
	if emu.sentMouse == nil {
		t.Fatalf("expected click forwarded to app")
	}
	if w.CopyModeActive() {
		t.Fatalf("expected copy mode not entered when mouse mode enabled")
	}

	// Shift bypass should start host selection instead of forwarding.
	emu.sentMouse = nil
	if !w.SendMouse(uv.MouseClickEvent{X: 2, Y: 1, Button: uv.MouseLeft, Mod: uv.ModShift}, MouseRouteAuto) {
		t.Fatalf("expected shift+click handled for host selection")
	}
	assertCopyModeInactive(t, w)
	if emu.sentMouse != nil {
		t.Fatalf("expected shift+click not forwarded to app")
	}

	emu.sentMouse = nil
	if !w.SendMouse(uv.MouseMotionEvent{X: 3, Y: 1, Button: uv.MouseNone, Mod: uv.ModShift}, MouseRouteAuto) {
		t.Fatalf("expected drag to start host selection")
	}
	if !w.CopyModeActive() {
		t.Fatalf("expected copy mode entered after drag threshold")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected host selection drag not forwarded to app")
	}
}

func TestMouseSelectionHostRouteInAltScreen(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 2,
		alt:  true,
		screen: [][]uv.Cell{
			mkCellsLine("A0", 5),
			mkCellsLine("A1", 5),
		},
	}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    2,
		updates: make(chan struct{}, 2),
	}
	w.updateMouseMode(ansi.ModeMouseX10, true)

	emu.sentMouse = nil
	if !w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteHostSelection) {
		t.Fatalf("expected click handled for host selection route")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected host selection click not forwarded to app")
	}

	if !w.SendMouse(uv.MouseMotionEvent{X: 3, Y: 0, Button: uv.MouseNone}, MouseRouteHostSelection) {
		t.Fatalf("expected drag motion handled for host selection route")
	}
	if !w.CopyModeActive() {
		t.Fatalf("expected copy mode entered in alt-screen for host selection")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected host selection drag not forwarded to app")
	}
}

func TestMouseClickDoesNotSelectWithoutDrag(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 1,
		screen: [][]uv.Cell{
			mkCellsLine("hello", 5),
		},
	}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    1,
		updates: make(chan struct{}, 2),
	}

	if !w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected click handled")
	}
	if !w.SendMouse(uv.MouseReleaseEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected release handled")
	}
	if w.CopyModeActive() {
		t.Fatalf("expected no copy mode for click without drag")
	}
}

func TestMouseWheelDoesNotGrowSelection(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 3,
		sb: [][]uv.Cell{
			mkCellsLine("S0", 5),
			mkCellsLine("S1", 5),
			mkCellsLine("S2", 5),
			mkCellsLine("S3", 5),
			mkCellsLine("S4", 5),
		},
		screen: [][]uv.Cell{
			mkCellsLine("A0", 5),
			mkCellsLine("A1", 5),
			mkCellsLine("A2", 5),
		},
	}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    3,
		updates: make(chan struct{}, 10),
	}

	orig := writeClipboard
	defer func() { writeClipboard = orig }()
	writeClipboard = func(string) error { return nil }

	if !w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected click handled")
	}
	if !w.SendMouse(uv.MouseMotionEvent{X: 3, Y: 2, Button: uv.MouseNone}, MouseRouteAuto) {
		t.Fatalf("expected motion handled")
	}
	if !w.SendMouse(uv.MouseReleaseEvent{X: 3, Y: 2, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected release handled")
	}
	if !w.CopySelectionFromMouseActive() {
		t.Fatalf("expected mouse selection active")
	}
	sx, sy, ex, ey := w.CopyMode.SelStartX, w.CopyMode.SelStartAbsY, w.CopyMode.SelEndX, w.CopyMode.SelEndAbsY

	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
		t.Fatalf("expected wheel handled")
	}
	if got := w.GetScrollbackOffset(); got != 3 {
		t.Fatalf("expected scrollback offset 3 after wheel, got %d", got)
	}
	if w.CopyMode.SelStartX != sx || w.CopyMode.SelStartAbsY != sy || w.CopyMode.SelEndX != ex || w.CopyMode.SelEndAbsY != ey {
		t.Fatalf("expected selection unchanged by wheel")
	}
}

func TestMouseDoubleClickSelectsWord(t *testing.T) {
	emu := &fakeEmu{
		cols: 11,
		rows: 1,
		screen: [][]uv.Cell{
			mkCellsLine("hello world", 11),
		},
	}
	clock := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)
	w := &Window{
		term:    emu,
		cols:    11,
		rows:    1,
		updates: make(chan struct{}, 2),
		mouseNow: func() time.Time {
			return clock
		},
	}

	if !w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected first click handled")
	}
	if !w.SendMouse(uv.MouseReleaseEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected first release handled")
	}
	clock = clock.Add(100 * time.Millisecond)
	if !w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
		t.Fatalf("expected second click handled")
	}
	if !w.CopyModeActive() || w.CopyMode == nil || !w.CopyMode.Selecting {
		t.Fatalf("expected word selection active")
	}
	if w.CopyMode.SelStartX != 0 || w.CopyMode.SelEndX != 4 {
		t.Fatalf("expected word bounds 0..4, got %d..%d", w.CopyMode.SelStartX, w.CopyMode.SelEndX)
	}
}

func TestMouseTripleClickSelectsLine(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 1,
		screen: [][]uv.Cell{
			mkCellsLine("hello", 5),
		},
	}
	clock := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    1,
		updates: make(chan struct{}, 2),
		mouseNow: func() time.Time {
			return clock
		},
	}

	for i := 0; i < 3; i++ {
		if !w.SendMouse(uv.MouseClickEvent{X: 2, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
			t.Fatalf("expected click %d handled", i+1)
		}
		if !w.SendMouse(uv.MouseReleaseEvent{X: 2, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto) {
			t.Fatalf("expected release %d handled", i+1)
		}
		clock = clock.Add(100 * time.Millisecond)
	}

	if !w.CopyModeActive() || w.CopyMode == nil || !w.CopyMode.Selecting {
		t.Fatalf("expected line selection active")
	}
	if w.CopyMode.SelStartX != 0 || w.CopyMode.SelEndX != 4 {
		t.Fatalf("expected line bounds 0..4, got %d..%d", w.CopyMode.SelStartX, w.CopyMode.SelEndX)
	}
}

func TestMouseShiftClickExtendsSelection(t *testing.T) {
	emu := &fakeEmu{
		cols: 11,
		rows: 1,
		screen: [][]uv.Cell{
			mkCellsLine("hello world", 11),
		},
	}
	clock := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)
	w := &Window{
		term:    emu,
		cols:    11,
		rows:    1,
		updates: make(chan struct{}, 2),
		mouseNow: func() time.Time {
			return clock
		},
	}

	// Double click selects "hello".
	_ = w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto)
	_ = w.SendMouse(uv.MouseReleaseEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto)
	clock = clock.Add(100 * time.Millisecond)
	_ = w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, MouseRouteAuto)

	if !w.CopyModeActive() || w.CopyMode == nil || !w.CopyMode.Selecting {
		t.Fatalf("expected selection active")
	}

	if !w.SendMouse(uv.MouseClickEvent{X: 8, Y: 0, Button: uv.MouseLeft, Mod: uv.ModShift}, MouseRouteAuto) {
		t.Fatalf("expected shift-click handled")
	}
	if w.CopyMode.SelEndX != 8 {
		t.Fatalf("expected selection end at 8, got %d", w.CopyMode.SelEndX)
	}
}
