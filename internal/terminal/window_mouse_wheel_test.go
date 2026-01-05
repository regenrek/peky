package terminal

import (
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
	if !w.SendMouse(click, MouseRouteAuto) {
		t.Fatalf("expected SendMouse to succeed")
	}
	if emu.sentMouse == nil {
		t.Fatalf("expected mouse event forwarded")
	}

	emu.sentMouse = nil
	w2 := &Window{term: emu}
	w2.updateMouseMode(ansi.ModeMouseX10, true)
	if w2.SendMouse(uv.MouseMotionEvent{X: 1, Y: 1}, MouseRouteAuto) {
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
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
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
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp, Mod: uv.ModShift}, MouseRouteAuto) {
		t.Fatalf("expected wheel+shift handled for scrollback")
	}
	if got := w.GetScrollbackOffset(); got != 1 {
		t.Fatalf("expected scrollback offset 1, got %d", got)
	}

	w.ExitScrollback()

	// Ctrl: page (rows-1) per tick. rows=3 => 2 lines.
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp, Mod: uv.ModCtrl}, MouseRouteAuto) {
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

	_ = w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto)
	if w.GetScrollbackOffset() == 0 {
		t.Fatalf("expected scrollback offset > 0 after wheel up")
	}
	if !w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback mode active after wheel up")
	}

	_ = w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelDown}, MouseRouteAuto)
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
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
		t.Fatalf("expected wheel handled in copy mode")
	}
	afterY := w.CopyMode.CursorAbsY
	if afterY != beforeY {
		t.Fatalf("expected copy cursor unchanged by wheel (from %d to %d)", beforeY, afterY)
	}
	if w.GetScrollbackOffset() != 3 {
		t.Fatalf("expected scrollback offset 3 after wheel, got %d", w.GetScrollbackOffset())
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected wheel not forwarded to app while in copy mode")
	}

	// Selecting should remain anchored while scrolling.
	w.CopyToggleSelect()
	if w.CopyMode == nil || !w.CopyMode.Selecting {
		t.Fatalf("expected copy selection enabled")
	}
	beforeStartX, beforeStartY := w.CopyMode.SelStartX, w.CopyMode.SelStartAbsY
	beforeEndX, beforeEndY := w.CopyMode.SelEndX, w.CopyMode.SelEndAbsY
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelDown}, MouseRouteAuto) {
		t.Fatalf("expected wheel handled in copy mode while selecting")
	}
	if w.CopyMode.SelStartX != beforeStartX || w.CopyMode.SelStartAbsY != beforeStartY || w.CopyMode.SelEndX != beforeEndX || w.CopyMode.SelEndAbsY != beforeEndY {
		t.Fatalf("expected selection bounds unchanged by wheel")
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
	if w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
		t.Fatalf("expected wheel ignored in alt-screen without mouse mode")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected no wheel forwarded without mouse mode")
	}

	// With mouse mode, wheel should be forwarded to the app.
	w.updateMouseMode(ansi.ModeMouseX10, true)
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
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
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
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
	if w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
		t.Fatalf("expected wheel unhandled when scrollback is empty and mouse mode disabled")
	}
	if w.GetScrollbackOffset() != 0 || w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback state unchanged, offset=%d mode=%v", w.GetScrollbackOffset(), w.ScrollbackModeActive())
	}
}

func TestMouseWheelNoScrollbackIgnoresMouseMode(t *testing.T) {
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
	if w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
		t.Fatalf("expected wheel ignored when scrollback is empty")
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected wheel not forwarded when scrollback is empty")
	}
	if w.GetScrollbackOffset() != 0 || w.ScrollbackModeActive() {
		t.Fatalf("expected no scrollback state changes, offset=%d mode=%v", w.GetScrollbackOffset(), w.ScrollbackModeActive())
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
	if !w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelUp}, MouseRouteAuto) {
		t.Fatalf("expected wheel handled in scrollback view even with mouse mode enabled")
	}
	if w.GetScrollbackOffset() != 3 {
		t.Fatalf("expected offset 3 after wheel up, got %d", w.GetScrollbackOffset())
	}
	if emu.sentMouse != nil {
		t.Fatalf("expected wheel not forwarded to app while in scrollback view")
	}

	emu.sentMouse = nil
	_ = w.SendMouse(uv.MouseWheelEvent{X: 0, Y: 0, Button: uv.MouseWheelDown}, MouseRouteAuto)
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
