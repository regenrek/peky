package terminal

import (
	"bytes"
	"image/color"
	"os"
	"runtime"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
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
	if emu.renderCalls != 1 {
		t.Fatalf("expected cached render after markDirty, got %d", emu.renderCalls)
	}
	w.refreshANSICache()
	if emu.renderCalls != 2 {
		t.Fatalf("expected refresh render after markDirty")
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
