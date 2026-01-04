package app

import (
	"context"
	"testing"
	"time"
)

func TestFocusSyncUpdatesDaemon(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "focus")
	if len(snap.Panes) == 0 {
		t.Fatalf("session snapshot missing panes")
	}
	paneSnap := snap.Panes[0]

	projectID := projectKey(snap.Path, "Proj")
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:   projectID,
		Name: "Proj",
		Path: snap.Path,
		Sessions: []SessionItem{{
			Name:       snap.Name,
			Status:     StatusRunning,
			ActivePane: paneSnap.Index,
			Panes: []PaneItem{{
				ID:     paneSnap.ID,
				Index:  paneSnap.Index,
				Title:  paneSnap.Title,
				Active: true,
			}},
		}},
	}}}

	m.applySelection(selectionState{ProjectID: projectID, Session: snap.Name, Pane: paneSnap.Index})
	cmd := m.consumeFocusSyncCmd()
	if cmd == nil {
		t.Fatalf("expected focus sync command")
	}
	_ = cmd()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := m.client.SnapshotState(ctx, 0)
	if err != nil {
		t.Fatalf("SnapshotState() error: %v", err)
	}
	if resp.FocusedPaneID != paneSnap.ID {
		t.Fatalf("FocusedPaneID = %q, want %q", resp.FocusedPaneID, paneSnap.ID)
	}
	if resp.FocusedSession != snap.Name {
		t.Fatalf("FocusedSession = %q, want %q", resp.FocusedSession, snap.Name)
	}
}

func TestFocusSyncCmdSurvivesClientReset(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "focus-reset")
	if len(snap.Panes) == 0 {
		t.Fatalf("session snapshot missing panes")
	}
	paneSnap := snap.Panes[0]

	projectID := projectKey(snap.Path, "Proj")
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:   projectID,
		Name: "Proj",
		Path: snap.Path,
		Sessions: []SessionItem{{
			Name:       snap.Name,
			Status:     StatusRunning,
			ActivePane: paneSnap.Index,
			Panes: []PaneItem{{
				ID:     paneSnap.ID,
				Index:  paneSnap.Index,
				Title:  paneSnap.Title,
				Active: true,
			}},
		}},
	}}}

	m.applySelection(selectionState{ProjectID: projectID, Session: snap.Name, Pane: paneSnap.Index})
	cmd := m.consumeFocusSyncCmd()
	if cmd == nil {
		t.Fatalf("expected focus sync command")
	}
	m.client = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("focus cmd panicked after client reset: %v", r)
		}
	}()
	_ = cmd()
}
