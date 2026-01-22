package sessiond

import (
	"context"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/terminal"
)

type focusManager struct {
	snapshot []native.SessionSnapshot
}

func (m *focusManager) SessionNames() []string { return nil }
func (m *focusManager) Snapshot(ctx context.Context, previewLines int) []native.SessionSnapshot {
	return m.snapshot
}
func (m *focusManager) Version() uint64 { return 0 }
func (m *focusManager) StartSession(context.Context, native.SessionSpec) (*native.Session, error) {
	return nil, nil
}
func (m *focusManager) KillSession(string) error                { return nil }
func (m *focusManager) RenameSession(string, string) error      { return nil }
func (m *focusManager) RenamePane(string, string, string) error { return nil }
func (m *focusManager) SplitPane(context.Context, string, string, bool, int) (string, error) {
	return "", nil
}
func (m *focusManager) ClosePane(context.Context, string, string) error { return nil }
func (m *focusManager) SwapPanes(string, string, string) error          { return nil }
func (m *focusManager) ResizePaneEdge(string, string, layout.ResizeEdge, int, bool, layout.SnapState) (layout.ApplyResult, error) {
	return layout.ApplyResult{}, nil
}
func (m *focusManager) ResetPaneSizes(string, string) (layout.ApplyResult, error) {
	return layout.ApplyResult{}, nil
}
func (m *focusManager) ZoomPane(string, string, bool) (layout.ApplyResult, error) {
	return layout.ApplyResult{}, nil
}
func (m *focusManager) SetPaneTool(string, string) error                { return nil }
func (m *focusManager) SetPaneBackground(string, int) error             { return nil }
func (m *focusManager) SendInput(context.Context, string, []byte) error { return nil }
func (m *focusManager) SendMouse(string, uv.MouseEvent, terminal.MouseRoute) error {
	return nil
}
func (m *focusManager) Window(string) paneWindow                          { return nil }
func (m *focusManager) PaneTags(string) ([]string, error)                 { return nil, nil }
func (m *focusManager) AddPaneTags(string, []string) ([]string, error)    { return nil, nil }
func (m *focusManager) RemovePaneTags(string, []string) ([]string, error) { return nil, nil }
func (m *focusManager) OutputSnapshot(string, int) ([]native.OutputLine, error) {
	return nil, nil
}
func (m *focusManager) OutputLinesSince(string, uint64) ([]native.OutputLine, uint64, bool, error) {
	return nil, 0, false, nil
}
func (m *focusManager) WaitForOutput(context.Context, string) bool { return false }
func (m *focusManager) SubscribeRawOutput(string, int) (<-chan native.OutputChunk, func(), error) {
	return nil, func() {}, nil
}
func (m *focusManager) PaneScrollbackSnapshot(string, int) (string, bool, error) {
	return "", false, nil
}
func (m *focusManager) SignalPane(string, string) error { return nil }
func (m *focusManager) Events() <-chan native.PaneEvent { return nil }
func (m *focusManager) Close()                          {}

func TestSetFocusSession(t *testing.T) {
	d := &Daemon{eventLog: newEventLog(10)}
	d.setFocusSession(" demo ")
	if d.focusedSession != "demo" {
		t.Fatalf("expected focusedSession demo, got %q", d.focusedSession)
	}
	if d.focusedPane != "" {
		t.Fatalf("expected focusedPane cleared")
	}
	events := d.eventLog.list(time.Time{}, time.Time{}, 10, nil)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventFocus || events[0].Session != "demo" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func TestSetFocusPaneResolvesSession(t *testing.T) {
	mgr := &focusManager{snapshot: []native.SessionSnapshot{{
		Name:  "s1",
		Panes: []native.PaneSnapshot{{ID: "p1"}},
	}}}
	d := &Daemon{eventLog: newEventLog(10), manager: mgr}
	d.setFocusPane("p1")
	if d.focusedPane != "p1" {
		t.Fatalf("expected focusedPane p1, got %q", d.focusedPane)
	}
	if d.focusedSession != "s1" {
		t.Fatalf("expected focusedSession s1, got %q", d.focusedSession)
	}
	events := d.eventLog.list(time.Time{}, time.Time{}, 10, nil)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].PaneID != "p1" || events[0].Session != "s1" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}
