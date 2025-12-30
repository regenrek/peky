package terminal

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func TestMouseModesAndSendMouse(t *testing.T) {
	emu := &fakeEmu{}
	w := &Window{term: emu}
	w.updateMouseMode(ansi.ModeMouseX10, true)
	if !w.HasMouseMode() {
		t.Fatalf("expected mouse mode enabled")
	}
	w.updateMouseMode(ansi.ModeMouseX10, false)
	if w.HasMouseMode() {
		t.Fatalf("expected mouse mode disabled")
	}
	w.updateMouseMode(ansi.ModeMouseButtonEvent, true)
	if !w.AllowsMouseMotion() {
		t.Fatalf("expected mouse motion allowed")
	}

	click := uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}
	if !w.SendMouse(click) {
		t.Fatalf("expected SendMouse to succeed")
	}
	if emu.sentMouse == nil {
		t.Fatalf("expected mouse event forwarded")
	}

	emu.sentMouse = nil
	w2 := &Window{term: emu}
	w2.updateMouseMode(ansi.ModeMouseX10, true)
	if w2.SendMouse(uv.MouseMotionEvent{X: 1, Y: 1}) {
		t.Fatalf("expected motion event blocked without motion mode")
	}
}

func TestMouseWheelScrollbackSteps(t *testing.T) {
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

	// Default: 3 lines per wheel tick.
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}) {
		t.Fatalf("expected wheel event handled for scrollback")
	}
	if got := w.GetScrollbackOffset(); got != 3 {
		t.Fatalf("expected scrollback offset 3, got %d", got)
	}
	if !w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback mode active")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected wheel not forwarded to app when mouse mode disabled")
	}

	w.ExitScrollback()
	if w.GetScrollbackOffset() != 0 || w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback exit")
	}

	// Shift: 1 line per tick.
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp, Mod: uv.ModShift}) {
		t.Fatalf("expected wheel+shift handled for scrollback")
	}
	if got := w.GetScrollbackOffset(); got != 1 {
		t.Fatalf("expected scrollback offset 1, got %d", got)
	}

	w.ExitScrollback()

	// Ctrl: page (rows-1) per tick. rows=3 => 2 lines.
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp, Mod: uv.ModCtrl}) {
		t.Fatalf("expected wheel+ctrl handled for scrollback")
	}
	if got := w.GetScrollbackOffset(); got != 2 {
		t.Fatalf("expected scrollback offset 2, got %d", got)
	}
}

func TestMouseWheelScrollbackAutoExitAtBottom(t *testing.T) {
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

	_ = w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp})
	if w.GetScrollbackOffset() == 0 {
		t.Fatalf("expected scrollback offset > 0 after wheel up")
	}
	if !w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback mode active after wheel up")
	}

	_ = w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelDown})
	if w.GetScrollbackOffset() != 0 {
		t.Fatalf("expected scrollback offset 0, got %d", w.GetScrollbackOffset())
	}
	if w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback mode to auto-exit at bottom")
	}
}

func TestMouseWheelCopyModeMovesCursor(t *testing.T) {
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
		cursor: uv.Pos(0, 1),
	}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    3,
		updates: make(chan struct{}, 10),
	}

	w.EnterCopyMode()
	if !w.CopyModeActive() || w.CopyMode == nil {
		t.Fatalf("expected copy mode active")
	}

	beforeY := w.CopyMode.CursorAbsY
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}) {
		t.Fatalf("expected wheel handled in copy mode")
	}
	afterY := w.CopyMode.CursorAbsY
	if afterY != beforeY-3 {
		t.Fatalf("expected copy cursor to move up by 3 (from %d to %d), got %d", beforeY, beforeY-3, afterY)
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected wheel not forwarded to app while in copy mode")
	}

	// Selecting should update the selection end as the cursor moves.
	w.CopyToggleSelect()
	if w.CopyMode == nil || !w.CopyMode.Selecting {
		t.Fatalf("expected copy selection enabled")
	}
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelDown}) {
		t.Fatalf("expected wheel handled in copy mode while selecting")
	}
	if w.CopyMode.SelEndAbsY != w.CopyMode.CursorAbsY {
		t.Fatalf("expected selection end to track cursor (selEnd=%d cursor=%d)", w.CopyMode.SelEndAbsY, w.CopyMode.CursorAbsY)
	}
}

func TestMouseWheelAltScreenForwarding(t *testing.T) {
	emu := &fakeEmu{}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    3,
		updates: make(chan struct{}, 2),
	}

	w.altScreen.Store(true)

	// Without mouse mode, wheel should be ignored in alt-screen.
	if w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}) {
		t.Fatalf("expected wheel ignored in alt-screen without mouse mode")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected no wheel forwarded without mouse mode")
	}

	// With mouse mode, wheel should be forwarded to the app.
	w.updateMouseMode(ansi.ModeMouseX10, true)
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}) {
		t.Fatalf("expected wheel forwarded in alt-screen when mouse mode enabled")
	}
	if emu.sentMouse == nil {
		t.Fatalf("expected wheel event forwarded to app")
	}
	if _, ok := emu.sentMouse.(uv.MouseWheelEvent); !ok {
		t.Fatalf("expected forwarded event to be MouseWheelEvent, got %T", emu.sentMouse)
	}
	if w.GetScrollbackOffset() != 0 {
		t.Fatalf("expected scrollback offset unchanged in alt-screen, got %d", w.GetScrollbackOffset())
	}
}

func TestMouseWheelScrollbackOverridesMouseMode(t *testing.T) {
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
		updates: make(chan struct{}, 2),
	}

	w.updateMouseMode(ansi.ModeMouseX10, true)
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}) {
		t.Fatalf("expected wheel handled for scrollback even when mouse mode enabled")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected wheel not forwarded in normal screen, got %T", emu.sentMouse)
	}
	if w.GetScrollbackOffset() != 3 || !w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback offset 3 and mode active, offset=%d mode=%v", w.GetScrollbackOffset(), w.ScrollbackModeActive())
	}
}

func TestMouseWheelNoScrollbackDoesNothing(t *testing.T) {
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
	if w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}) {
		t.Fatalf("expected wheel unhandled when scrollback is empty and mouse mode disabled")
	}
	if w.GetScrollbackOffset() != 0 || w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback state unchanged, offset=%d mode=%v", w.GetScrollbackOffset(), w.ScrollbackModeActive())
	}
}

func assertSendMouseHandled(t *testing.T, w *Window, event uv.MouseEvent, msg string) {
	t.Helper()
	if !w.SendMouse(event) {
		t.Fatalf(msg)
	}
}

func assertCopyModeInactive(t *testing.T, w *Window) {
	t.Helper()
	if w.CopyModeActive() {
		t.Fatalf("expected copy mode inactive")
	}
}

func assertCopyModeActive(t *testing.T, w *Window) {
	t.Helper()
	if !w.CopyModeActive() || w.CopyMode == nil {
		t.Fatalf("expected copy mode active")
	}
}

func assertCopySelecting(t *testing.T, w *Window) {
	t.Helper()
	if w.CopyMode == nil || !w.CopyMode.Selecting {
		t.Fatalf("expected selecting true")
	}
}

func assertMouseSelectionActive(t *testing.T, w *Window) {
	t.Helper()
	if !w.CopySelectionFromMouseActive() {
		t.Fatalf("expected mouse selection active")
	}
}

func assertSelectionCursor(t *testing.T, w *Window, x, absY int) {
	t.Helper()
	if w.CopyMode == nil || w.CopyMode.CursorX != x || w.CopyMode.CursorAbsY != absY {
		t.Fatalf("expected cursor at (%d,%d), got (%d,%d)", x, absY, w.CopyMode.CursorX, w.CopyMode.CursorAbsY)
	}
}

func assertSelectionStartEnd(t *testing.T, w *Window, sx, sy, ex, ey int) {
	t.Helper()
	if w.CopyMode == nil || w.CopyMode.SelStartX != sx || w.CopyMode.SelStartAbsY != sy || w.CopyMode.SelEndX != ex || w.CopyMode.SelEndAbsY != ey {
		t.Fatalf("expected selection start(%d,%d) end(%d,%d), got start(%d,%d) end(%d,%d)",
			sx, sy, ex, ey, w.CopyMode.SelStartX, w.CopyMode.SelStartAbsY, w.CopyMode.SelEndX, w.CopyMode.SelEndAbsY)
	}
}

func assertMouseNotForwarded(t *testing.T, emu *fakeEmu) {
	t.Helper()
	if emu.sentMouse != nil {
		t.Fatalf("expected mouse not forwarded to app")
	}
}

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
	assertSendMouseHandled(t, w, uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}, "expected click to start selection")
	assertCopyModeActive(t, w)
	assertCopySelecting(t, w)
	assertMouseSelectionActive(t, w)
	// sbLen=5, offset=0 => topAbsY=5, y=0 => absY=5
	assertSelectionCursor(t, w, 1, 5)
	assertSelectionStartEnd(t, w, 1, 5, 1, 5)
	assertMouseNotForwarded(t, emu)

	emu.sentMouse = nil
	assertSendMouseHandled(t, w, uv.MouseMotionEvent{X: 3, Y: 2, Button: uv.MouseNone}, "expected drag motion to update selection")
	// y=2 => absY=7
	assertSelectionCursor(t, w, 3, 7)
	assertSelectionStartEnd(t, w, 1, 5, 3, 7)
	assertMouseNotForwarded(t, emu)

	emu.sentMouse = nil
	assertSendMouseHandled(t, w, uv.MouseReleaseEvent{X: 3, Y: 2, Button: uv.MouseLeft}, "expected release handled in copy mode")
	assertMouseSelectionActive(t, w)
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

	if !w.SendMouse(uv.MouseClickEvent{X: 0, Y: 0, Button: uv.MouseLeft}) {
		t.Fatalf("expected click to start selection")
	}
	if !w.SendMouse(uv.MouseMotionEvent{X: 4, Y: 0, Button: uv.MouseNone}) {
		t.Fatalf("expected motion to extend selection")
	}
	if !w.SendMouse(uv.MouseReleaseEvent{X: 4, Y: 0, Button: uv.MouseLeft}) {
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
	if !w.SendMouse(uv.MouseClickEvent{X: 1, Y: 0, Button: uv.MouseLeft}) {
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
	if !w.SendMouse(uv.MouseClickEvent{X: 2, Y: 1, Button: uv.MouseLeft, Mod: uv.ModShift}) {
		t.Fatalf("expected shift+click handled for host selection")
	}
	if !w.CopyModeActive() {
		t.Fatalf("expected copy mode entered with shift+click bypass")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected shift+click not forwarded to app")
	}
}

func TestMouseWheelScrollbackViewWithMouseModeEnabled(t *testing.T) {
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

	w.updateMouseMode(ansi.ModeMouseX10, true)
	w.EnterScrollback()
	if !w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback mode active")
	}

	emu.sentMouse = nil
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}) {
		t.Fatalf("expected wheel handled in scrollback view even with mouse mode enabled")
	}
	if w.GetScrollbackOffset() != 3 {
		t.Fatalf("expected offset 3 after wheel up, got %d", w.GetScrollbackOffset())
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected wheel not forwarded to app while in scrollback view")
	}

	emu.sentMouse = nil
	_ = w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelDown})
	if w.GetScrollbackOffset() != 0 {
		t.Fatalf("expected offset back to 0, got %d", w.GetScrollbackOffset())
	}
	if w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback auto-exit at bottom after wheel down")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected wheel not forwarded to app while in scrollback view")
	}
}
