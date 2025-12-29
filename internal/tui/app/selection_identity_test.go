package app

import "testing"

func TestHandleDashboardSnapshotUsesTabResolverOnStale(t *testing.T) {
	m := newTestModelLite()
	m.tab = TabDashboard
	m.selectionVersion = 10
	m.selection = selectionState{
		ProjectID: projectKey("/alpha", "Alpha"),
		Session:   "beta-1",
		Pane:      "1",
	}

	snapshot := dashboardSnapshotMsg{Result: dashboardSnapshotResult{
		Data:     DashboardData{Projects: sampleProjects()},
		Settings: m.settings,
		Version:  9,
	}}
	m.handleDashboardSnapshot(snapshot)

	if m.selection.ProjectID != projectKey("/beta", "Beta") {
		t.Fatalf("selection.ProjectID = %q, want %q", m.selection.ProjectID, projectKey("/beta", "Beta"))
	}
	if m.selection.Session != "beta-1" {
		t.Fatalf("selection.Session = %q, want %q", m.selection.Session, "beta-1")
	}
}

func TestResolveSelectionByProjectIDWithNameCollision(t *testing.T) {
	groups := []ProjectGroup{
		{
			ID:   projectKey("/one", "Same"),
			Name: "Same",
			Path: "/one",
			Sessions: []SessionItem{
				{Name: "s1"},
			},
		},
		{
			ID:   projectKey("/two", "Same"),
			Name: "Same",
			Path: "/two",
			Sessions: []SessionItem{
				{Name: "s2"},
			},
		},
	}

	selA := resolveSelection(groups, selectionState{ProjectID: groups[0].ID, Session: "s1"})
	if selA.ProjectID != groups[0].ID {
		t.Fatalf("selection for first project = %#v", selA)
	}

	selB := resolveSelection(groups, selectionState{ProjectID: groups[1].ID, Session: "s2"})
	if selB.ProjectID != groups[1].ID {
		t.Fatalf("selection for second project = %#v", selB)
	}
}
