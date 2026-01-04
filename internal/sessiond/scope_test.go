package sessiond

import (
	"context"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/regenrek/peakypanes/internal/native"
)

type fakeScopeManager struct {
	snapshot []native.SessionSnapshot
}

func (m *fakeScopeManager) SessionNames() []string { return nil }
func (m *fakeScopeManager) Snapshot(ctx context.Context, previewLines int) []native.SessionSnapshot {
	return m.snapshot
}
func (m *fakeScopeManager) Version() uint64 { return 0 }
func (m *fakeScopeManager) StartSession(context.Context, native.SessionSpec) (*native.Session, error) {
	return nil, nil
}
func (m *fakeScopeManager) RestoreSession(context.Context, native.SessionRestoreSpec) (*native.Session, error) {
	return nil, nil
}
func (m *fakeScopeManager) KillSession(string) error                { return nil }
func (m *fakeScopeManager) RenameSession(string, string) error      { return nil }
func (m *fakeScopeManager) RenamePane(string, string, string) error { return nil }
func (m *fakeScopeManager) SplitPane(context.Context, string, string, bool, int) (string, error) {
	return "", nil
}
func (m *fakeScopeManager) ClosePane(context.Context, string, string) error         { return nil }
func (m *fakeScopeManager) SwapPanes(string, string, string) error                  { return nil }
func (m *fakeScopeManager) SetPaneTool(string, string) error                        { return nil }
func (m *fakeScopeManager) SendInput(context.Context, string, []byte) error         { return nil }
func (m *fakeScopeManager) SendMouse(string, uv.MouseEvent) error                   { return nil }
func (m *fakeScopeManager) Window(string) paneWindow                                { return nil }
func (m *fakeScopeManager) PaneTags(string) ([]string, error)                       { return nil, nil }
func (m *fakeScopeManager) AddPaneTags(string, []string) ([]string, error)          { return nil, nil }
func (m *fakeScopeManager) RemovePaneTags(string, []string) ([]string, error)       { return nil, nil }
func (m *fakeScopeManager) OutputSnapshot(string, int) ([]native.OutputLine, error) { return nil, nil }
func (m *fakeScopeManager) OutputLinesSince(string, uint64) ([]native.OutputLine, uint64, bool, error) {
	return nil, 0, false, nil
}
func (m *fakeScopeManager) WaitForOutput(context.Context, string) bool { return false }
func (m *fakeScopeManager) SubscribeRawOutput(string, int) (<-chan native.OutputChunk, func(), error) {
	return nil, func() {}, nil
}
func (m *fakeScopeManager) PaneScrollbackSnapshot(string, int) (string, bool, error) {
	return "", false, nil
}
func (m *fakeScopeManager) SignalPane(string, string) error { return nil }
func (m *fakeScopeManager) Events() <-chan native.PaneEvent { return nil }
func (m *fakeScopeManager) Close()                          {}

func TestResolveScopeTargetsErrors(t *testing.T) {
	d := &Daemon{}
	if _, err := d.resolveScopeTargets(""); err == nil {
		t.Fatalf("expected error for empty scope")
	}
	d.manager = &fakeScopeManager{}
	if _, err := d.resolveScopeTargets("all"); err == nil {
		t.Fatalf("expected error for empty sessions")
	}
}

func TestResolveScopeTargetsAllAndSession(t *testing.T) {
	sessions := []native.SessionSnapshot{
		{Name: "s1", Path: "/proj/a", Panes: []native.PaneSnapshot{{ID: "p1"}, {ID: "p2"}}},
		{Name: "s2", Path: "/proj/b", Panes: []native.PaneSnapshot{{ID: "p3"}}},
	}
	d := &Daemon{manager: &fakeScopeManager{snapshot: sessions}}
	ids, err := d.resolveScopeTargets("all")
	if err != nil {
		t.Fatalf("resolve all: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}
	d.focusedSession = "s2"
	ids, err = d.resolveScopeTargets("session")
	if err != nil {
		t.Fatalf("resolve session: %v", err)
	}
	if len(ids) != 1 || ids[0] != "p3" {
		t.Fatalf("unexpected session ids: %+v", ids)
	}
}

func TestResolveScopeTargetsProject(t *testing.T) {
	sessions := []native.SessionSnapshot{
		{Name: "s1", Path: "/proj/a", Panes: []native.PaneSnapshot{{ID: "p1"}}},
		{Name: "s2", Path: "/proj/a", Panes: []native.PaneSnapshot{{ID: "p2"}}},
		{Name: "s3", Path: "/proj/b", Panes: []native.PaneSnapshot{{ID: "p3"}}},
	}
	d := &Daemon{manager: &fakeScopeManager{snapshot: sessions}, focusedSession: "s1"}
	ids, err := d.resolveScopeTargets("project")
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
}

func TestResolveScopeTargetsProjectSinglePath(t *testing.T) {
	sessions := []native.SessionSnapshot{
		{Name: "s1", Path: "relative/path", Panes: []native.PaneSnapshot{{ID: "p1"}}},
	}
	d := &Daemon{manager: &fakeScopeManager{snapshot: sessions}}
	ids, err := d.resolveScopeTargets("project")
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	if len(ids) != 1 || ids[0] != "p1" {
		t.Fatalf("unexpected ids: %+v", ids)
	}
}

func TestCollectHelpers(t *testing.T) {
	sessions := []native.SessionSnapshot{
		{Name: "s1", Panes: []native.PaneSnapshot{{ID: "p1"}, {ID: "p1"}, {ID: ""}}},
		{Name: "s2", Panes: []native.PaneSnapshot{{ID: "p2"}}},
	}
	all := collectPaneIDs(sessions)
	if len(all) != 2 {
		t.Fatalf("expected 2 unique pane ids, got %d", len(all))
	}
	sessionIDs := collectSessionPaneIDs(sessions, "missing")
	if sessionIDs != nil {
		t.Fatalf("expected nil for missing session")
	}
}
