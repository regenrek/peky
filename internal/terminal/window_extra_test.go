package terminal

import (
	"bytes"
	"image/color"
	"os"
	"runtime"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestLooksLikeCPR(t *testing.T) {
	if !looksLikeCPR([]byte("\x1b[1;2R")) {
		t.Fatalf("expected CPR detection")
	}
	if looksLikeCPR([]byte("\x1b[12R")) {
		t.Fatalf("expected CPR detection to fail without semicolon")
	}
	if looksLikeCPR([]byte("not-cpr")) {
		t.Fatalf("expected CPR detection to fail for random data")
	}
}

func TestTranslateCPR(t *testing.T) {
	emu := &fakeEmu{cursor: uv.Pos(3, 4)}
	w := &Window{term: emu}
	out := w.translateCPR(emu, []byte("\x1b[10;20R"))
	if string(out) != "\x1b[5;4R" {
		t.Fatalf("translateCPR = %q", string(out))
	}
	raw := []byte("hello")
	if string(w.translateCPR(emu, raw)) != string(raw) {
		t.Fatalf("expected non-CPR to pass through")
	}
}

func TestWriteToPTY(t *testing.T) {
	var buf bytes.Buffer
	w := &Window{}
	w.writeToPTY(&buf, []byte("hi"))
	if buf.String() != "hi" {
		t.Fatalf("writeToPTY = %q", buf.String())
	}
}

func TestResizeClampsState(t *testing.T) {
	emu := &fakeEmu{
		cols: 4,
		rows: 3,
		sb: [][]uv.Cell{
			mkCellsLine("S0", 4),
			mkCellsLine("S1", 4),
			mkCellsLine("S2", 4),
		},
		screen: [][]uv.Cell{
			mkCellsLine("A0", 4),
			mkCellsLine("A1", 4),
			mkCellsLine("A2", 4),
		},
	}
	w := &Window{
		term:    emu,
		cols:    4,
		rows:    3,
		updates: make(chan struct{}, 1),
	}
	w.ScrollbackOffset = 10
	w.ScrollbackMode = true
	if err := w.Resize(5, 2); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	if emu.cols != 5 || emu.rows != 2 {
		t.Fatalf("expected emulator resize to 5x2, got %dx%d", emu.cols, emu.rows)
	}
	if w.ScrollbackOffset > emu.ScrollbackLen() {
		t.Fatalf("expected scrollback offset clamped, got %d", w.ScrollbackOffset)
	}
}

func TestResizeErrors(t *testing.T) {
	var w *Window
	if err := w.Resize(10, 10); err == nil {
		t.Fatalf("expected nil window error")
	}
	w = &Window{}
	w.closed.Store(true)
	if err := w.Resize(10, 10); err == nil {
		t.Fatalf("expected closed window error")
	}
}

func TestScrollAndCopyModes(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 3,
		sb: [][]uv.Cell{
			mkCellsLine("S0", 5),
			mkCellsLine("S1", 5),
			mkCellsLine("S2", 5),
		},
		screen: [][]uv.Cell{
			mkCellsLine("A0", 5),
			mkCellsLine("A1", 5),
			mkCellsLine("A2", 5),
		},
		cursor: uv.Pos(1, 1),
	}
	w := &Window{
		term:    emu,
		cols:    5,
		rows:    3,
		updates: make(chan struct{}, 5),
	}
	if w.CopyModeActive() {
		t.Fatalf("expected copy mode inactive")
	}
	w.EnterScrollback()
	if !w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback mode active")
	}
	w.ScrollUp(2)
	if w.GetScrollbackOffset() != 2 {
		t.Fatalf("expected scrollback offset 2, got %d", w.GetScrollbackOffset())
	}
	w.PageUp()
	if w.GetScrollbackOffset() <= 2 {
		t.Fatalf("expected page up to increase offset")
	}
	w.ScrollDown(1)
	if w.GetScrollbackOffset() != 2 {
		t.Fatalf("expected scrollback offset 2, got %d", w.GetScrollbackOffset())
	}
	w.PageDown()
	if w.GetScrollbackOffset() != 0 || w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback to exit at bottom")
	}

	w.EnterCopyMode()
	if !w.CopyModeActive() {
		t.Fatalf("expected copy mode active")
	}
	before := w.CopyMode.CursorAbsY
	w.CopyPageUp()
	if w.CopyMode.CursorAbsY == before {
		t.Fatalf("expected copy cursor to move")
	}
	w.CopyPageDown()
	w.ExitCopyMode()
	if w.CopyModeActive() {
		t.Fatalf("expected copy mode inactive")
	}
	w.ScrollUp(1)
	w.ScrollToBottom()
	if w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback mode inactive after ScrollToBottom")
	}
}

func TestWindowAccessors(t *testing.T) {
	w := &Window{id: "pane1", cols: 80, rows: 24, updates: make(chan struct{}, 1)}
	w.title.Store("title")
	if w.ID() != "pane1" {
		t.Fatalf("ID = %q", w.ID())
	}
	if w.Title() != "title" {
		t.Fatalf("Title = %q", w.Title())
	}
	w.SetTitle("next")
	if w.Title() != "next" {
		t.Fatalf("SetTitle = %q", w.Title())
	}
	w.exited.Store(true)
	w.exitStatus.Store(7)
	if !w.Exited() || w.ExitStatus() != 7 {
		t.Fatalf("Exited/ExitStatus = %v/%d", w.Exited(), w.ExitStatus())
	}
	if w.Cols() != 80 || w.Rows() != 24 {
		t.Fatalf("Cols/Rows = %d/%d", w.Cols(), w.Rows())
	}
	if w.Updates() == nil {
		t.Fatalf("Updates channel nil")
	}
}

func TestEnvHelpers(t *testing.T) {
	if runtimeGOOS() != runtime.GOOS {
		t.Fatalf("runtimeGOOS mismatch")
	}

	prevShell := os.Getenv("SHELL")
	t.Setenv("SHELL", "/bin/customshell")
	if got := detectShell(); got != "/bin/customshell" {
		t.Fatalf("detectShell = %q", got)
	}
	t.Setenv("SHELL", "")
	if got := detectShell(); strings.TrimSpace(got) == "" {
		t.Fatalf("detectShell fallback empty")
	}
	if prevShell != "" {
		_ = os.Setenv("SHELL", prevShell)
	}

	env := []string{"A=1", "B=2"}
	merged := mergeEnv(env, []string{"B=3", "C=4"})
	if !hasEnv(merged, "A") || !hasEnv(merged, "B") || !hasEnv(merged, "C") {
		t.Fatalf("mergeEnv result = %#v", merged)
	}
	if hasEnv(merged, "") {
		t.Fatalf("expected hasEnv false for empty key")
	}
	if envKey("Key=Value") != "KEY" {
		t.Fatalf("envKey unexpected")
	}
	if envKey("novalue") != "" {
		t.Fatalf("expected empty envKey for invalid input")
	}
}

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

func TestMouseWheelForwardedWhenMouseModeEnabled(t *testing.T) {
	emu := &fakeEmu{
		cols: 5,
		rows: 3,
		sb: [][]uv.Cell{
			mkCellsLine("S0", 5),
			mkCellsLine("S1", 5),
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
		t.Fatalf("expected wheel forwarded when mouse mode enabled")
	}
	if emu.sentMouse == nil {
		t.Fatalf("expected wheel event forwarded to app")
	}
	if w.GetScrollbackOffset() != 0 || w.ScrollbackModeActive() {
		t.Fatalf("expected scrollback unchanged when mouse mode enabled, offset=%d mode=%v", w.GetScrollbackOffset(), w.ScrollbackModeActive())
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

func TestViewANSIAndRenderHelpers(t *testing.T) {
	emu := &fakeEmu{renderValue: "hi"}
	w := &Window{term: emu, cacheDirty: true}
	if got := w.ViewANSI(); got != "hi" {
		t.Fatalf("ViewANSI = %q", got)
	}
	if emu.renderCalls != 1 {
		t.Fatalf("expected 1 render call, got %d", emu.renderCalls)
	}
	if got := w.ViewANSI(); got != "hi" {
		t.Fatalf("ViewANSI cache = %q", got)
	}
	if emu.renderCalls != 1 {
		t.Fatalf("expected cached render, got %d", emu.renderCalls)
	}
	w.markDirty()
	_ = w.ViewANSI()
	if emu.renderCalls != 2 {
		t.Fatalf("expected render after markDirty")
	}

	cm := &CopyMode{
		Active:       true,
		Selecting:    true,
		CursorX:      0,
		CursorAbsY:   0,
		SelStartX:    0,
		SelStartAbsY: 0,
		SelEndX:      1,
		SelEndAbsY:   0,
	}
	hl := selectionHighlighter(0, cm)
	cursor, selected := hl(0, 0)
	if !cursor || !selected {
		t.Fatalf("expected cursor+selection at origin")
	}
	if _, selected := hl(2, 0); selected {
		t.Fatalf("expected selection false outside range")
	}

	cell := uv.Cell{
		Content: "A",
		Width:   1,
		Style: uv.Style{
			Fg: color.RGBA{R: 1, G: 2, B: 3, A: 255},
		},
	}
	term := &renderTerm{cell: cell, cursor: uv.Pos(0, 0)}
	out := RenderEmulatorLipgloss(term, 1, 1, RenderOptions{ShowCursor: true, Profile: termenv.TrueColor})
	if !strings.Contains(out, "A") {
		t.Fatalf("expected rendered cell, got %q", out)
	}
}

type renderTerm struct {
	cell   uv.Cell
	cursor uv.Position
}

func (r *renderTerm) CellAt(int, int) *uv.Cell { return &r.cell }
func (r *renderTerm) CursorPosition() uv.Position {
	return r.cursor
}
