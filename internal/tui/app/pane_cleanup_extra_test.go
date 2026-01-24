package app

import (
	"context"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
)

func TestPaneCleanupHelpers(t *testing.T) {
	dead, live := splitDeadPanes([]PaneItem{{ID: "p1", Dead: true}, {ID: "p2"}})
	if len(dead) != 1 || len(live) != 1 {
		t.Fatalf("expected split dead/live")
	}
	anchor := selectCleanupAnchor([]PaneItem{{ID: "p2"}, {ID: "p3", Active: true}})
	if anchor == nil || anchor.ID != "p3" {
		t.Fatalf("expected active anchor")
	}

	projects := []ProjectGroup{{
		ID:   "p",
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:  "sess",
			Path:  "/tmp",
			Panes: []PaneItem{{ID: "p1", Dead: true}},
		}},
	}}
	targets, hasLive := collectOfflineSessions(projects)
	if hasLive {
		t.Fatalf("expected no live panes")
	}
	if len(targets) != 1 {
		t.Fatalf("expected one target")
	}

	m := newTestModelLite()
	m.client = nil
	cmd := m.cleanupDeadPanes()
	if cmd != nil {
		t.Fatalf("expected nil cmd when client missing")
	}

	m.handlePaneCleanup(paneCleanupMsg{Noop: "noop"})
	if m.toast.Text == "" {
		t.Fatalf("expected toast for noop")
	}
	m.handlePaneCleanup(paneCleanupMsg{Err: "boom"})
	if m.toast.Text == "" {
		t.Fatalf("expected toast for error")
	}
}

func TestPaneCleanupRunWithClient(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "cleanup")
	if len(snap.Panes) == 0 {
		t.Fatalf("expected panes in snapshot")
	}

	settings, err := defaultDashboardConfig(layout.DashboardConfig{})
	if err != nil {
		t.Fatalf("defaultDashboardConfig error: %v", err)
	}
	result := buildDashboardData(dashboardSnapshotInput{
		Selection: selectionState{Session: snap.Name, Pane: snap.Panes[0].Index},
		Tab:       TabProject,
		Config:    &layout.Config{Projects: []layout.ProjectConfig{{Name: "App", Session: snap.Name, Path: snap.Path}}},
		Settings:  settings,
		Version:   1,
		Sessions:  []native.SessionSnapshot{snap},
	})
	m.data = result.Data
	m.selection = result.Resolved

	session := &SessionItem{Name: "cleanup-restart", Path: snap.Path, LayoutName: snap.LayoutName}
	cmd := m.restartSessionCmd(session)
	if cmd == nil {
		t.Fatalf("expected restart cmd")
	}
	msg := cmd()
	cleanupMsg, ok := msg.(paneCleanupMsg)
	if !ok {
		t.Fatalf("expected paneCleanupMsg")
	}
	if cleanupMsg.Err != "" {
		t.Fatalf("unexpected restart error: %s", cleanupMsg.Err)
	}

	anchorIndex := snap.Panes[0].Index
	resultMsg := m.cleanupPanesRun(snap.Name, anchorIndex, true, []PaneItem{{ID: "missing"}})
	if resultMsg.Added == 0 && resultMsg.Failed == 0 {
		t.Fatalf("expected cleanup activity")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = m.client.SnapshotState(ctx, 0)
}
