package app

import (
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
)

func TestStartSessionNativeClientMissing(t *testing.T) {
	m := newTestModelLite()
	path := t.TempDir()

	cmd := m.startSessionNative("sess", path, "", true)
	if cmd == nil {
		t.Fatalf("expected session start cmd")
	}
	msg := cmd()
	started, ok := msg.(sessionStartedMsg)
	if !ok {
		t.Fatalf("expected sessionStartedMsg")
	}
	if started.Err == nil {
		t.Fatalf("expected error when client missing")
	}
}

func TestStartProjectNativeValidationAndDetached(t *testing.T) {
	m := newTestModelLite()

	if cmd := m.startProjectNative(SessionItem{Name: "sess"}, false); cmd != nil {
		t.Fatalf("expected nil cmd for missing path")
	}

	path := t.TempDir()
	cmd := m.startProjectNative(SessionItem{Name: "sess", Path: path}, false)
	if cmd == nil {
		t.Fatalf("expected start cmd for valid path")
	}

	if cmd := m.startSessionAtPathDetached(filepath.Join(path, "missing")); cmd != nil {
		t.Fatalf("expected nil cmd for invalid path")
	}
	if cmd := m.startSessionAtPathDetached(path); cmd != nil {
		t.Fatalf("expected nil cmd when client missing")
	}
}

func TestClosePaneAndSwapPaneErrors(t *testing.T) {
	m := newTestModelLite()

	if cmd := m.closePane("", "", ""); cmd != nil {
		t.Fatalf("expected nil cmd for empty selection")
	}
	if cmd := m.closePane("missing", "1", ""); cmd != nil {
		t.Fatalf("expected nil cmd for missing session")
	}

	m.data.Projects[0].Sessions[0].Status = StatusStopped
	if cmd := m.swapPaneWith(PaneSwapChoice{PaneIndex: "2"}); cmd != nil {
		t.Fatalf("expected nil cmd for stopped session")
	}
}

func TestClosePaneSuccess(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "app")
	if len(snap.Panes) == 0 {
		t.Fatalf("expected snapshot panes")
	}

	cfg := &layout.Config{Projects: []layout.ProjectConfig{{Name: "App", Session: snap.Name, Path: snap.Path}}}
	settings, err := defaultDashboardConfig(layout.DashboardConfig{})
	if err != nil {
		t.Fatalf("defaultDashboardConfig() error: %v", err)
	}
	result := buildDashboardData(dashboardSnapshotInput{
		Selection: selectionState{Session: snap.Name, Pane: snap.Panes[0].Index},
		Tab:       TabProject,
		Config:    cfg,
		Settings:  settings,
		Version:   1,
		Sessions:  []native.SessionSnapshot{snap},
	})
	m.data = result.Data
	m.selection = result.Resolved

	cmd := m.closePane(snap.Name, snap.Panes[0].Index, snap.Panes[0].ID)
	if cmd == nil {
		t.Fatalf("expected close pane cmd")
	}
	if m.selection.Pane != "" {
		t.Fatalf("expected pane selection cleared")
	}
}
