package sessiond

import (
	"context"
	"testing"

	"github.com/muesli/termenv"
)

type fakeTerminalWindow struct {
	altScreen      bool
	copyMode       bool
	copySelecting  bool
	mouseSelection bool
	scrollback     bool
	scrollOffset   int
	yankText       string
	viewANSI       string
	viewLipgloss   string
	hasMouse       bool
	allowMotion    bool
	updateSeq      uint64
	resizeCols     int
	resizeRows     int
	calls          map[string]int
	lastCopyMoveX  int
	lastCopyMoveY  int
}

func (f *fakeTerminalWindow) record(name string) {
	if f.calls == nil {
		f.calls = make(map[string]int)
	}
	f.calls[name]++
}

func (f *fakeTerminalWindow) UpdateSeq() uint64 { return f.updateSeq }
func (f *fakeTerminalWindow) ANSICacheSeq() uint64 {
	return f.updateSeq
}

func (f *fakeTerminalWindow) CopyModeActive() bool      { return f.copyMode }
func (f *fakeTerminalWindow) CopySelectionActive() bool { return f.copyMode && f.copySelecting }
func (f *fakeTerminalWindow) CopySelectionFromMouseActive() bool {
	return f.copyMode && f.copySelecting && f.mouseSelection
}
func (f *fakeTerminalWindow) Resize(cols, rows int) error {
	f.resizeCols = cols
	f.resizeRows = rows
	f.record("resize")
	return nil
}
func (f *fakeTerminalWindow) CopyMove(dx, dy int) {
	f.lastCopyMoveX = dx
	f.lastCopyMoveY = dy
	f.record("copyMove")
}
func (f *fakeTerminalWindow) CopyPageDown() { f.record("copyPageDown") }
func (f *fakeTerminalWindow) CopyPageUp()   { f.record("copyPageUp") }
func (f *fakeTerminalWindow) CopyToggleSelect() {
	f.copySelecting = !f.copySelecting
	f.mouseSelection = false
	f.record("copyToggle")
}
func (f *fakeTerminalWindow) CopyYankText() string { f.record("copyYank"); return f.yankText }
func (f *fakeTerminalWindow) EnterCopyMode()       { f.copyMode = true; f.record("enterCopy") }
func (f *fakeTerminalWindow) EnterScrollback()     { f.scrollback = true; f.record("enterScrollback") }
func (f *fakeTerminalWindow) ExitCopyMode() {
	f.copyMode = false
	f.copySelecting = false
	f.mouseSelection = false
	f.record("exitCopy")
}
func (f *fakeTerminalWindow) ExitScrollback() {
	f.scrollback = false
	f.scrollOffset = 0
	f.record("exitScrollback")
}
func (f *fakeTerminalWindow) GetScrollbackOffset() int   { return f.scrollOffset }
func (f *fakeTerminalWindow) IsAltScreen() bool          { return f.altScreen }
func (f *fakeTerminalWindow) PageDown()                  { f.record("pageDown") }
func (f *fakeTerminalWindow) PageUp()                    { f.record("pageUp") }
func (f *fakeTerminalWindow) ScrollDown(lines int)       { f.record("scrollDown") }
func (f *fakeTerminalWindow) ScrollToBottom()            { f.record("scrollBottom") }
func (f *fakeTerminalWindow) ScrollToTop()               { f.record("scrollTop") }
func (f *fakeTerminalWindow) ScrollUp(lines int)         { f.record("scrollUp") }
func (f *fakeTerminalWindow) ScrollbackModeActive() bool { return f.scrollback }
func (f *fakeTerminalWindow) ViewLipglossCtx(ctx context.Context, showCursor bool, profile termenv.Profile) (string, error) {
	f.record("viewLipgloss")
	return f.viewLipgloss, nil
}
func (f *fakeTerminalWindow) ViewANSICtx(ctx context.Context) (string, error) {
	f.record("viewANSI")
	return f.viewANSI, nil
}
func (f *fakeTerminalWindow) ViewLipgloss(showCursor bool, profile termenv.Profile) string {
	f.record("viewLipgloss")
	return f.viewLipgloss
}
func (f *fakeTerminalWindow) ViewANSI() string {
	f.record("viewANSI")
	return f.viewANSI
}
func (f *fakeTerminalWindow) HasMouseMode() bool {
	return f.hasMouse
}
func (f *fakeTerminalWindow) AllowsMouseMotion() bool {
	return f.allowMotion
}

func TestHandleAltScreenKeyResetsModes(t *testing.T) {
	win := &fakeTerminalWindow{
		altScreen:    true,
		copyMode:     true,
		scrollback:   true,
		scrollOffset: 2,
	}
	resp, handled := handleAltScreenKey(win)
	if !handled || resp.Handled {
		t.Fatalf("expected handled alt screen response")
	}
	if win.copyMode || win.scrollback || win.scrollOffset != 0 {
		t.Fatalf("expected modes cleared")
	}
	if win.calls["exitCopy"] == 0 || win.calls["exitScrollback"] == 0 {
		t.Fatalf("expected exit calls, got %#v", win.calls)
	}
}

func TestHandleCopyModeKeyPaths(t *testing.T) {
	win := &fakeTerminalWindow{copyMode: true, yankText: "yanked"}

	resp, handled := handleCopyModeKey(win, "y")
	if !handled || !resp.Handled || resp.YankText != "yanked" {
		t.Fatalf("expected yank handled, resp=%#v", resp)
	}
	if win.copyMode {
		t.Fatalf("expected copy mode exit")
	}

	win.copyMode = true
	resp, handled = handleCopyModeKey(win, "v")
	if !handled || resp.Toast == "" {
		t.Fatalf("expected toggle select toast, resp=%#v", resp)
	}

	win.copyMode = true
	resp, handled = handleCopyModeKey(win, "esc")
	if !handled || !resp.Handled || resp.Toast == "" {
		t.Fatalf("expected escape handled, resp=%#v", resp)
	}
}

func TestHandleScrollbackKeyPaths(t *testing.T) {
	win := &fakeTerminalWindow{scrollback: true, scrollOffset: 1}

	resp, handled := handleScrollbackKey(win, TerminalKeyRequest{CopyToggle: true})
	if !handled || !resp.Handled || resp.Toast == "" {
		t.Fatalf("expected copy toggle handled, resp=%#v", resp)
	}
	if win.calls["enterCopy"] == 0 {
		t.Fatalf("expected enter copy mode")
	}

	resp, handled = handleScrollbackKey(win, TerminalKeyRequest{ScrollbackToggle: true})
	if !handled || !resp.Handled {
		t.Fatalf("expected scrollback toggle handled, resp=%#v", resp)
	}
	if win.calls["pageUp"] == 0 {
		t.Fatalf("expected page up")
	}

	resp, handled = handleScrollbackKey(win, TerminalKeyRequest{Key: "g"})
	if !handled || !resp.Handled {
		t.Fatalf("expected home handled, resp=%#v", resp)
	}
	if win.calls["scrollTop"] == 0 {
		t.Fatalf("expected scroll top")
	}
}

func TestHandleNormalKeyPaths(t *testing.T) {
	win := &fakeTerminalWindow{}
	resp := handleNormalKey(win, TerminalKeyRequest{ScrollbackToggle: true})
	if !resp.Handled || resp.Toast == "" || !win.scrollback {
		t.Fatalf("expected scrollback start, resp=%#v", resp)
	}
	if win.calls["pageUp"] == 0 {
		t.Fatalf("expected page up")
	}

	win = &fakeTerminalWindow{}
	resp = handleNormalKey(win, TerminalKeyRequest{CopyToggle: true})
	if !resp.Handled || resp.Toast == "" || !win.copyMode {
		t.Fatalf("expected copy mode start, resp=%#v", resp)
	}
}

func TestHandleCopyModeKeyMovement(t *testing.T) {
	win := &fakeTerminalWindow{copyMode: true}
	resp, handled := handleCopyModeKey(win, "up")
	if !handled || !resp.Handled || win.calls["copyMove"] == 0 {
		t.Fatalf("expected copy move up, resp=%#v calls=%#v", resp, win.calls)
	}
	win.copyMode = true
	resp, handled = handleCopyModeKey(win, "pgdown")
	if !handled || !resp.Handled || win.calls["copyPageDown"] == 0 {
		t.Fatalf("expected copy page down, resp=%#v calls=%#v", resp, win.calls)
	}
}

func TestHandleScrollbackKeyMovement(t *testing.T) {
	win := &fakeTerminalWindow{scrollback: true, scrollOffset: 1}
	resp, handled := handleScrollbackKey(win, TerminalKeyRequest{Key: "down"})
	if !handled || !resp.Handled || win.calls["scrollDown"] == 0 {
		t.Fatalf("expected scroll down, resp=%#v calls=%#v", resp, win.calls)
	}
	resp, handled = handleScrollbackKey(win, TerminalKeyRequest{Key: "pgdown"})
	if !handled || !resp.Handled || win.calls["pageDown"] == 0 {
		t.Fatalf("expected page down, resp=%#v calls=%#v", resp, win.calls)
	}
}

func TestHandleScrollbackKeyExit(t *testing.T) {
	win := &fakeTerminalWindow{scrollback: true, scrollOffset: 2}
	resp, handled := handleScrollbackKey(win, TerminalKeyRequest{Key: "esc"})
	if !handled || !resp.Handled || win.calls["exitScrollback"] == 0 {
		t.Fatalf("expected scrollback exit, resp=%#v calls=%#v", resp, win.calls)
	}
}

func TestHandleTerminalKeyNormal(t *testing.T) {
	win := &fakeTerminalWindow{}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	resp, err := d.handleTerminalKey(TerminalKeyRequest{PaneID: "pane-1", ScrollbackToggle: true})
	if err != nil {
		t.Fatalf("handleTerminalKey: %v", err)
	}
	if !resp.Handled || win.calls["enterScrollback"] == 0 {
		t.Fatalf("expected scrollback entered, resp=%#v calls=%#v", resp, win.calls)
	}
}

func TestHandleCopyModeKeyAutoExitSelectionPassthrough(t *testing.T) {
	win := &fakeTerminalWindow{copyMode: true, copySelecting: true, mouseSelection: true}

	resp, handled := handleCopyModeKey(win, "a")
	if !handled || resp.Handled {
		t.Fatalf("expected passthrough when selection active, resp=%#v handled=%v", resp, handled)
	}
	if win.calls["exitCopy"] == 0 || win.copyMode {
		t.Fatalf("expected copy mode exited, calls=%#v copyMode=%v", win.calls, win.copyMode)
	}
}

func TestHandleCopyModeKeySelectionKeepsCopyModeForCopyKeys(t *testing.T) {
	win := &fakeTerminalWindow{copyMode: true, copySelecting: true}

	resp, handled := handleCopyModeKey(win, "j")
	if !handled || !resp.Handled {
		t.Fatalf("expected movement handled, resp=%#v handled=%v", resp, handled)
	}
	if win.calls["exitCopy"] != 0 {
		t.Fatalf("did not expect exit copy mode, calls=%#v", win.calls)
	}
	if win.calls["copyMove"] == 0 {
		t.Fatalf("expected copyMove, calls=%#v", win.calls)
	}
}

func TestHandleCopyModeKeyMouseSelectionPassesPrintableKeys(t *testing.T) {
	win := &fakeTerminalWindow{copyMode: true, copySelecting: true, mouseSelection: true, yankText: "yanked"}

	resp, handled := handleCopyModeKey(win, "y")
	if !handled || resp.Handled {
		t.Fatalf("expected y passthrough for mouse selection, resp=%#v handled=%v", resp, handled)
	}
	if win.calls["copyYank"] != 0 {
		t.Fatalf("did not expect yank for mouse selection, calls=%#v", win.calls)
	}
	if win.calls["exitCopy"] == 0 || win.copyMode {
		t.Fatalf("expected copy mode exited, calls=%#v copyMode=%v", win.calls, win.copyMode)
	}
}

func TestHandleCopyModeKeyMouseSelectionCopyShortcutsYank(t *testing.T) {
	win := &fakeTerminalWindow{copyMode: true, copySelecting: true, mouseSelection: true, yankText: "yanked"}

	resp, handled := handleCopyModeKey(win, "ctrl+c")
	if !handled || !resp.Handled || resp.YankText != "yanked" {
		t.Fatalf("expected copy shortcut to yank, resp=%#v handled=%v", resp, handled)
	}
	if win.calls["copyYank"] == 0 {
		t.Fatalf("expected yank call, calls=%#v", win.calls)
	}
	if win.calls["exitCopy"] == 0 || win.copyMode {
		t.Fatalf("expected copy mode exited, calls=%#v copyMode=%v", win.calls, win.copyMode)
	}
}

func TestHandleCopyModeKeyNoSelectionStillConsumesNonCopyKeys(t *testing.T) {
	win := &fakeTerminalWindow{copyMode: true}

	resp, handled := handleCopyModeKey(win, "a")
	if !handled || !resp.Handled {
		t.Fatalf("expected key consumed when no selection, resp=%#v handled=%v", resp, handled)
	}
	if win.calls["exitCopy"] != 0 {
		t.Fatalf("did not expect exit copy mode, calls=%#v", win.calls)
	}
}
