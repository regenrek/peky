package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestRefreshCmdWithClient(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "app")
	m.selection = selectionState{Session: snap.Name}

	cmd := m.refreshCmd(1)
	msg := cmd()
	snapshot, ok := msg.(dashboardSnapshotMsg)
	if !ok {
		t.Fatalf("expected dashboardSnapshotMsg")
	}
	if snapshot.Result.Err != nil {
		t.Fatalf("refresh error: %v", snapshot.Result.Err)
	}
	if len(snapshot.Result.Data.Projects) == 0 {
		t.Fatalf("expected projects in snapshot")
	}
}

func TestFetchPaneViewsCmdError(t *testing.T) {
	m := newTestModel(t)

	cmd := m.fetchPaneViewsCmd([]sessiond.PaneViewRequest{{
		PaneID: "missing",
		Cols:   10,
		Rows:   4,
		Mode:   sessiond.PaneViewANSI,
	}})
	if cmd == nil {
		t.Fatalf("expected pane views cmd")
	}
	msg := cmd()
	viewMsg, ok := msg.(paneViewsMsg)
	if !ok {
		t.Fatalf("expected paneViewsMsg")
	}
	if viewMsg.Err == nil {
		t.Fatalf("expected error for missing pane view")
	}
}
