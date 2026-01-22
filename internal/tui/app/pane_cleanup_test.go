package app

import "testing"

func TestSplitDeadPanes(t *testing.T) {
	panes := []PaneItem{
		{ID: "p1"},
		{ID: "p2", Dead: true},
		{ID: "p3", Disconnected: true},
	}
	dead, live := splitDeadPanes(panes)
	if len(dead) != 2 || len(live) != 1 {
		t.Fatalf("dead=%d live=%d", len(dead), len(live))
	}
}

func TestSelectCleanupAnchor(t *testing.T) {
	panes := []PaneItem{
		{ID: "p1"},
		{ID: "p2", Active: true},
	}
	if anchor := selectCleanupAnchor(panes); anchor == nil || anchor.ID != "p2" {
		t.Fatalf("expected active pane anchor")
	}
	if anchor := selectCleanupAnchor([]PaneItem{{ID: "p1"}}); anchor == nil || anchor.ID != "p1" {
		t.Fatalf("expected first pane anchor")
	}
	if anchor := selectCleanupAnchor(nil); anchor != nil {
		t.Fatalf("expected nil anchor")
	}
}

func TestCollectOfflineSessions(t *testing.T) {
	projects := sampleProjects()
	for i := range projects[0].Sessions[0].Panes {
		projects[0].Sessions[0].Panes[i].Dead = true
	}
	targets, hasLive := collectOfflineSessions(projects)
	if len(targets) != 1 || targets[0].name != "alpha-1" {
		t.Fatalf("targets=%#v", targets)
	}
	if !hasLive {
		t.Fatalf("expected hasLive true")
	}

	for projIdx := range projects {
		for sessIdx := range projects[projIdx].Sessions {
			for paneIdx := range projects[projIdx].Sessions[sessIdx].Panes {
				projects[projIdx].Sessions[sessIdx].Panes[paneIdx].Dead = true
			}
		}
	}
	targets, hasLive = collectOfflineSessions(projects)
	if len(targets) == 0 || hasLive {
		t.Fatalf("expected offline targets without live panes")
	}
}

func TestHandlePaneCleanup(t *testing.T) {
	m := newTestModelLite()
	m.handlePaneCleanup(paneCleanupMsg{Noop: "No dead/offline panes"})
	if m.toast.Level != toastInfo || m.toast.Text == "" {
		t.Fatalf("expected info toast, got %#v", m.toast)
	}

	m = newTestModelLite()
	cmd := m.handlePaneCleanup(paneCleanupMsg{Err: "boom"})
	if m.toast.Level != toastError || m.toast.Text == "" {
		t.Fatalf("expected error toast, got %#v", m.toast)
	}
	if cmd == nil {
		t.Fatalf("expected refresh command")
	}

	m = newTestModelLite()
	m.handlePaneCleanup(paneCleanupMsg{Restarted: 1, Sessions: []string{"alpha-1"}})
	if m.toast.Level != toastSuccess || m.toast.Text == "" {
		t.Fatalf("expected success toast, got %#v", m.toast)
	}
}
