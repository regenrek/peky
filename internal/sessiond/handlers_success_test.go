package sessiond

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/termframe"
	"github.com/regenrek/peakypanes/internal/terminal"
)

type fakeManager struct {
	windowID             string
	window               paneWindow
	sessions             []string
	snapshot             []native.SessionSnapshot
	version              uint64
	events               chan native.PaneEvent
	lastSnapshotPreview  int
	lastSnapshotDeadline time.Time
	lastInput            []byte
	inputs               [][]byte
	lastMouse            uv.MouseEvent
	mouseCalls           int
	lastMouseRoute       terminal.MouseRoute
	lastKilled           string
	lastRename           [2]string
	lastSwap             [3]string
	lastTool             [2]string
	lastBackground       struct {
		paneID     string
		background int
	}
	lastResize struct {
		sessionName string
		paneID      string
		edge        layout.ResizeEdge
		delta       int
		snap        bool
		snapState   layout.SnapState
	}
	lastReset struct {
		sessionName string
		paneID      string
	}
	lastZoom struct {
		sessionName string
		paneID      string
		toggle      bool
	}
}

func (m *fakeManager) SessionNames() []string { return m.sessions }
func (m *fakeManager) Snapshot(ctx context.Context, previewLines int) []native.SessionSnapshot {
	m.lastSnapshotPreview = previewLines
	if ctx != nil {
		if deadline, ok := ctx.Deadline(); ok {
			m.lastSnapshotDeadline = deadline
		}
	}
	return m.snapshot
}
func (m *fakeManager) Version() uint64 { return m.version }
func (m *fakeManager) StartSession(context.Context, native.SessionSpec) (*native.Session, error) {
	return &native.Session{Name: "demo"}, nil
}
func (m *fakeManager) KillSession(name string) error {
	m.lastKilled = name
	return nil
}
func (m *fakeManager) RenameSession(oldName, newName string) error {
	m.lastRename = [2]string{oldName, newName}
	return nil
}
func (m *fakeManager) RenamePane(sessionName, paneIndex, newTitle string) error {
	return nil
}
func (m *fakeManager) SplitPane(context.Context, string, string, bool, int) (string, error) {
	return "2", nil
}
func (m *fakeManager) ClosePane(context.Context, string, string) error {
	return nil
}
func (m *fakeManager) SwapPanes(sessionName, paneA, paneB string) error {
	m.lastSwap = [3]string{sessionName, paneA, paneB}
	return nil
}
func (m *fakeManager) ResizePaneEdge(sessionName, paneID string, edge layout.ResizeEdge, delta int, snap bool, snapState layout.SnapState) (layout.ApplyResult, error) {
	m.lastResize.sessionName = sessionName
	m.lastResize.paneID = paneID
	m.lastResize.edge = edge
	m.lastResize.delta = delta
	m.lastResize.snap = snap
	m.lastResize.snapState = snapState
	return layout.ApplyResult{Changed: true, Affected: []string{paneID}}, nil
}
func (m *fakeManager) ResetPaneSizes(sessionName, paneID string) (layout.ApplyResult, error) {
	m.lastReset.sessionName = sessionName
	m.lastReset.paneID = paneID
	return layout.ApplyResult{Changed: true}, nil
}
func (m *fakeManager) ZoomPane(sessionName, paneID string, toggle bool) (layout.ApplyResult, error) {
	m.lastZoom.sessionName = sessionName
	m.lastZoom.paneID = paneID
	m.lastZoom.toggle = toggle
	return layout.ApplyResult{Changed: true}, nil
}
func (m *fakeManager) SetPaneTool(paneID, tool string) error {
	m.lastTool = [2]string{paneID, tool}
	return nil
}
func (m *fakeManager) SetPaneBackground(paneID string, background int) error {
	m.lastBackground.paneID = paneID
	m.lastBackground.background = background
	return nil
}
func (m *fakeManager) SendInput(_ context.Context, paneID string, input []byte) error {
	m.lastInput = append([]byte(nil), input...)
	m.inputs = append(m.inputs, append([]byte(nil), input...))
	return nil
}
func (m *fakeManager) SendMouse(paneID string, event uv.MouseEvent, route terminal.MouseRoute) error {
	m.lastMouse = event
	m.mouseCalls++
	m.lastMouseRoute = route
	return nil
}
func (m *fakeManager) Window(paneID string) paneWindow {
	if paneID != m.windowID {
		return nil
	}
	return m.window
}
func (m *fakeManager) PaneTags(string) ([]string, error)                       { return nil, nil }
func (m *fakeManager) AddPaneTags(string, []string) ([]string, error)          { return nil, nil }
func (m *fakeManager) RemovePaneTags(string, []string) ([]string, error)       { return nil, nil }
func (m *fakeManager) OutputSnapshot(string, int) ([]native.OutputLine, error) { return nil, nil }
func (m *fakeManager) OutputLinesSince(string, uint64) ([]native.OutputLine, uint64, bool, error) {
	return nil, 0, false, nil
}
func (m *fakeManager) WaitForOutput(context.Context, string) bool { return false }
func (m *fakeManager) SubscribeRawOutput(string, int) (<-chan native.OutputChunk, func(), error) {
	ch := make(chan native.OutputChunk)
	close(ch)
	return ch, func() {}, nil
}
func (m *fakeManager) PaneScrollbackSnapshot(string, int) (string, bool, error) {
	return "", false, nil
}
func (m *fakeManager) SignalPane(string, string) error { return nil }
func (m *fakeManager) Events() <-chan native.PaneEvent {
	return m.events
}
func (m *fakeManager) Close() {
	if m.events != nil {
		close(m.events)
	}
}

func TestHandlePaneViewSuccess(t *testing.T) {
	win := &fakeTerminalWindow{
		viewFrame:   termframe.Frame{Cols: 1, Rows: 1, Cells: []termframe.Cell{{Content: "lip", Width: 1}}},
		hasMouse:    true,
		allowMotion: true,
	}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(PaneViewRequest{
		PaneID: "pane-1",
		Cols:   0,
		Rows:   0,
	})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err := d.handlePaneView(payload)
	if err != nil {
		t.Fatalf("handlePaneView: %v", err)
	}
	var resp PaneViewResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decodePayload: %v", err)
	}
	if resp.PaneID != "pane-1" || len(resp.Frame.Cells) == 0 || resp.Frame.Cells[0].Content != "lip" {
		t.Fatalf("unexpected pane view response: %#v", resp)
	}
	if win.resizeCols != 1 || win.resizeRows != 1 {
		t.Fatalf("expected resize to 1x1, got %dx%d", win.resizeCols, win.resizeRows)
	}
	if !resp.HasMouse || !resp.AllowMotion {
		t.Fatalf("expected mouse flags true")
	}
}

func TestHandleSetPaneToolSuccess(t *testing.T) {
	manager := &fakeManager{}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(SetPaneToolRequest{PaneID: "pane-1", Tool: "codex"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSetPaneTool(payload); err != nil {
		t.Fatalf("handleSetPaneTool: %v", err)
	}
	if manager.lastTool != [2]string{"pane-1", "codex"} {
		t.Fatalf("unexpected tool update: %#v", manager.lastTool)
	}
}

func TestHandleResizePaneSuccess(t *testing.T) {
	win := &fakeTerminalWindow{}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(ResizePaneRequest{SessionName: "alpha", PaneID: "pane-1", Edge: ResizeEdgeRight, Delta: 12, Snap: true})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleResizePane(payload); err != nil {
		t.Fatalf("handleResizePane: %v", err)
	}
	if manager.lastResize.sessionName != "alpha" || manager.lastResize.paneID != "pane-1" || manager.lastResize.edge != layout.ResizeEdgeRight || manager.lastResize.delta != 12 || !manager.lastResize.snap {
		t.Fatalf("unexpected resize call: %#v", manager.lastResize)
	}
}

func TestHandleResetPaneSizesSuccess(t *testing.T) {
	manager := &fakeManager{}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(ResetSizesRequest{SessionName: "alpha", PaneID: "pane-1"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleResetPaneSizes(payload); err != nil {
		t.Fatalf("handleResetPaneSizes: %v", err)
	}
	if manager.lastReset.sessionName != "alpha" || manager.lastReset.paneID != "pane-1" {
		t.Fatalf("unexpected reset call: %#v", manager.lastReset)
	}
}

func TestHandleZoomPaneSuccess(t *testing.T) {
	manager := &fakeManager{}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(ZoomPaneRequest{SessionName: "alpha", PaneID: "pane-1", Toggle: true})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleZoomPane(payload); err != nil {
		t.Fatalf("handleZoomPane: %v", err)
	}
	if manager.lastZoom.sessionName != "alpha" || manager.lastZoom.paneID != "pane-1" || !manager.lastZoom.toggle {
		t.Fatalf("unexpected zoom call: %#v", manager.lastZoom)
	}
}

func TestHandleSessionNamesSnapshotAndRename(t *testing.T) {
	manager := &fakeManager{
		sessions: []string{"alpha", "beta"},
		snapshot: []native.SessionSnapshot{{Name: "alpha"}},
		version:  7,
		windowID: "pane-1",
		window:   &fakeTerminalWindow{},
	}
	d := &Daemon{manager: manager}

	assertSessionNames(t, d)
	assertSnapshot(t, d, manager)
	assertRenameSession(t, d, manager)
}

func assertSessionNames(t *testing.T, d *Daemon) {
	t.Helper()
	data, err := d.handleSessionNames()
	if err != nil {
		t.Fatalf("handleSessionNames: %v", err)
	}
	var namesResp SessionNamesResponse
	if err := decodePayload(data, &namesResp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(namesResp.Names) != 2 {
		t.Fatalf("expected 2 session names")
	}
}

func assertSnapshot(t *testing.T, d *Daemon, manager *fakeManager) {
	t.Helper()
	payload, err := encodePayload(SnapshotRequest{PreviewLines: 2})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err := d.handleSnapshot(payload)
	if err != nil {
		t.Fatalf("handleSnapshot: %v", err)
	}
	var snapResp SnapshotResponse
	if err := decodePayload(data, &snapResp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if snapResp.Version != 7 || len(snapResp.Sessions) != 1 {
		t.Fatalf("unexpected snapshot response: %#v", snapResp)
	}
	if manager.lastSnapshotPreview != 2 {
		t.Fatalf("snapshot preview=%d want 2", manager.lastSnapshotPreview)
	}
	if manager.lastSnapshotDeadline.IsZero() {
		t.Fatalf("expected snapshot deadline to be set")
	}
}

func assertRenameSession(t *testing.T, d *Daemon, manager *fakeManager) {
	t.Helper()
	payload, err := encodePayload(RenameSessionRequest{OldName: "alpha", NewName: "gamma"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err := d.handleRenameSession(payload)
	if err != nil {
		t.Fatalf("handleRenameSession: %v", err)
	}
	var renameResp RenameSessionResponse
	if err := decodePayload(data, &renameResp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if renameResp.NewName != "gamma" || manager.lastRename != [2]string{"alpha", "gamma"} {
		t.Fatalf("unexpected rename response: %#v", renameResp)
	}
}

func TestHandleKillSplitCloseSwap(t *testing.T) {
	manager := &fakeManager{windowID: "pane-1", window: &fakeTerminalWindow{}}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(KillSessionRequest{Name: "alpha"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleKillSession(payload); err != nil {
		t.Fatalf("handleKillSession: %v", err)
	}
	if manager.lastKilled != "alpha" {
		t.Fatalf("expected kill recorded")
	}

	payload, err = encodePayload(SplitPaneRequest{SessionName: "alpha", PaneIndex: "1", Vertical: true, Percent: 30})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err := d.handleSplitPane(payload)
	if err != nil {
		t.Fatalf("handleSplitPane: %v", err)
	}
	var splitResp SplitPaneResponse
	if err := decodePayload(data, &splitResp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if splitResp.NewIndex == "" {
		t.Fatalf("expected split response index")
	}

	payload, err = encodePayload(ClosePaneRequest{SessionName: "alpha", PaneIndex: "1"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleClosePane(payload); err != nil {
		t.Fatalf("handleClosePane: %v", err)
	}

	payload, err = encodePayload(SwapPanesRequest{SessionName: "alpha", PaneA: "1", PaneB: "2"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSwapPanes(payload); err != nil {
		t.Fatalf("handleSwapPanes: %v", err)
	}
	if manager.lastSwap != [3]string{"alpha", "1", "2"} {
		t.Fatalf("expected swap recorded")
	}
}

func TestHandleTerminalPayloads(t *testing.T) {
	win := &fakeTerminalWindow{altScreen: true, copyMode: true, scrollback: true, scrollOffset: 1}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(TerminalActionRequest{PaneID: "pane-1", Action: TerminalPageUp})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err := d.handleTerminalActionPayload(payload)
	if err != nil {
		t.Fatalf("handleTerminalActionPayload: %v", err)
	}
	var actionResp TerminalActionResponse
	if err := decodePayload(data, &actionResp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if actionResp.PaneID != "pane-1" || win.calls["pageUp"] == 0 {
		t.Fatalf("expected terminal action handled")
	}

	payload, err = encodePayload(TerminalKeyRequest{PaneID: "pane-1", Key: "esc"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err = d.handleHandleKey(payload)
	if err != nil {
		t.Fatalf("handleHandleKey: %v", err)
	}
	var keyResp TerminalKeyResponse
	if err := decodePayload(data, &keyResp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if keyResp.Handled {
		t.Fatalf("expected handled false for alt screen reset")
	}
}

func TestHandleRenamePaneSuccess(t *testing.T) {
	manager := &fakeManager{windowID: "pane-1", window: &fakeTerminalWindow{}}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(RenamePaneRequest{SessionName: "alpha", PaneIndex: "1", NewTitle: "new-title"})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleRenamePane(payload); err != nil {
		t.Fatalf("handleRenamePane: %v", err)
	}
}

func TestHandleTerminalKeyCopyMode(t *testing.T) {
	win := &fakeTerminalWindow{copyMode: true}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	resp, err := d.handleTerminalKey(TerminalKeyRequest{PaneID: "pane-1", Key: "y"})
	if err != nil {
		t.Fatalf("handleTerminalKey: %v", err)
	}
	if !resp.Handled || win.calls["copyYank"] == 0 {
		t.Fatalf("expected copy yank handled, resp=%#v calls=%#v", resp, win.calls)
	}
}

func TestHandleStartSessionSuccess(t *testing.T) {
	dir := t.TempDir()
	config := []byte("session: demo\nlayout:\n  name: demo\n  panes:\n    - cmd: \"echo hi\"\n")
	if err := os.WriteFile(filepath.Join(dir, ".peky.yml"), config, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	manager := &fakeManager{windowID: "pane-1", window: &fakeTerminalWindow{}}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(StartSessionRequest{Name: "demo", Path: dir})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	data, err := d.handleStartSession(payload)
	if err != nil {
		t.Fatalf("handleStartSession: %v", err)
	}
	var resp StartSessionResponse
	if err := decodePayload(data, &resp); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if resp.Name != "demo" || resp.Path != dir {
		t.Fatalf("unexpected start response: %#v", resp)
	}
}

func TestHandleSendInputAndMouseSuccess(t *testing.T) {
	manager := &fakeManager{windowID: "pane-1", window: &fakeTerminalWindow{}}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(SendInputRequest{PaneID: "pane-1", Input: []byte("hi")})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendInput(payload); err != nil {
		t.Fatalf("handleSendInput: %v", err)
	}
	if string(manager.lastInput) != "hi" {
		t.Fatalf("expected input forwarded")
	}

	payload, err = encodePayload(SendMouseRequest{PaneID: "pane-1", Event: MouseEventPayload{X: 1, Y: 2, Button: 1, Action: MouseActionPress}})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendMouse(payload); err != nil {
		t.Fatalf("handleSendMouse: %v", err)
	}
	if manager.lastMouse == nil {
		t.Fatalf("expected mouse forwarded")
	}
}

func TestHandleSendMouseWheelCountSuccess(t *testing.T) {
	manager := &fakeManager{windowID: "pane-1", window: &fakeTerminalWindow{}}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(SendMouseRequest{
		PaneID: "pane-1",
		Event: MouseEventPayload{
			X:          1,
			Y:          2,
			Button:     4, // wheel up
			Wheel:      true,
			WheelCount: 7,
			Route:      MouseRouteAuto,
		},
	})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendMouse(payload); err != nil {
		t.Fatalf("handleSendMouse: %v", err)
	}
	if manager.mouseCalls != 7 {
		t.Fatalf("mouseCalls=%d", manager.mouseCalls)
	}
	if _, ok := manager.lastMouse.(uv.MouseWheelEvent); !ok {
		t.Fatalf("expected wheel event, got %T", manager.lastMouse)
	}
}

func TestHandleSendMouseWheelCountDefaultsToOne(t *testing.T) {
	manager := &fakeManager{windowID: "pane-1", window: &fakeTerminalWindow{}}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(SendMouseRequest{
		PaneID: "pane-1",
		Event: MouseEventPayload{
			X:          1,
			Y:          2,
			Button:     4,
			Wheel:      true,
			WheelCount: 0,
			Route:      MouseRouteAuto,
		},
	})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendMouse(payload); err != nil {
		t.Fatalf("handleSendMouse: %v", err)
	}
	if manager.mouseCalls != 1 {
		t.Fatalf("mouseCalls=%d", manager.mouseCalls)
	}
}

func TestHandleSendMouseWheelCountClamped(t *testing.T) {
	manager := &fakeManager{windowID: "pane-1", window: &fakeTerminalWindow{}}
	d := &Daemon{manager: manager}

	payload, err := encodePayload(SendMouseRequest{
		PaneID: "pane-1",
		Event: MouseEventPayload{
			X:          1,
			Y:          2,
			Button:     4,
			Wheel:      true,
			WheelCount: 999999,
			Route:      MouseRouteAuto,
		},
	})
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if _, err := d.handleSendMouse(payload); err != nil {
		t.Fatalf("handleSendMouse: %v", err)
	}
	if manager.mouseCalls != sessiondMouseWheelCountMax {
		t.Fatalf("mouseCalls=%d", manager.mouseCalls)
	}
}

func TestHandleTerminalActionAndKeySuccess(t *testing.T) {
	win := &fakeTerminalWindow{altScreen: true, copyMode: true, scrollback: true, scrollOffset: 1}
	manager := &fakeManager{windowID: "pane-1", window: win}
	d := &Daemon{manager: manager}

	resp, err := d.terminalAction(TerminalActionRequest{PaneID: "pane-1", Action: TerminalPageDown})
	if err != nil {
		t.Fatalf("terminalAction: %v", err)
	}
	if resp.PaneID != "pane-1" || win.calls["pageDown"] == 0 {
		t.Fatalf("expected page down handled, resp=%#v calls=%#v", resp, win.calls)
	}

	keyResp, err := d.handleTerminalKey(TerminalKeyRequest{PaneID: "pane-1", Key: "esc"})
	if err != nil {
		t.Fatalf("handleTerminalKey: %v", err)
	}
	if keyResp.Handled {
		t.Fatalf("expected handled false for alt screen reset")
	}
	if win.copyMode || win.scrollback || win.scrollOffset != 0 {
		t.Fatalf("expected modes cleared")
	}
}
