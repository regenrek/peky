package app

import "testing"

func TestDashboardPaneHits(t *testing.T) {
	m := newTestModelLite()
	m.state = StateDashboard
	m.tab = TabDashboard
	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}

	hits := m.dashboardPaneHits()
	if len(hits) == 0 {
		t.Fatalf("expected dashboard hits")
	}
	if hits[0].Selection.ProjectID == "" || hits[0].Selection.Session == "" {
		t.Fatalf("unexpected hit selection: %#v", hits[0].Selection)
	}

	hit := hits[0]
	found, ok := m.hitTestPane(hit.Outer.X+1, hit.Outer.Y+1)
	if !ok || found.PaneID != hit.PaneID {
		t.Fatalf("expected hitTestPane match")
	}

	headerHits := m.headerHitRects()
	if len(headerHits) == 0 {
		t.Fatalf("expected header hits")
	}
	_, ok = m.hitTestHeader(headerHits[0].Rect.X, headerHits[0].Rect.Y)
	if !ok {
		t.Fatalf("expected header hit")
	}
}

func TestProjectPaneHitsLayout(t *testing.T) {
	m := newTestModelLite()
	m.state = StateDashboard
	m.tab = TabProject
	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}

	layoutHits := m.projectPaneHits()
	if len(layoutHits) != len(m.selectedSession().Panes) {
		t.Fatalf("expected layout hits for panes")
	}
	if layoutHits[0].Outer.Empty() {
		t.Fatalf("expected outer rect for layout hit")
	}
}

func TestPaneViewRequestsAndLookup(t *testing.T) {
	m := newTestModelLite()
	m.state = StateDashboard
	m.tab = TabProject
	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}

	hit, ok := m.paneHitFor("p1")
	if !ok {
		t.Fatalf("expected pane hit for p1")
	}
	req := m.paneViewRequestForHit(hit)
	if req == nil || req.PaneID != "p1" {
		t.Fatalf("expected pane view request, got %#v", req)
	}

	key := paneViewKey{
		PaneID: "p1",
		Cols:   req.Cols,
		Rows:   req.Rows,
	}
	m.paneViews[key] = paneViewEntry{rendered: map[bool]string{false: "view"}}
	if got := m.paneView("p1", req.Cols, req.Rows, false); got != "view" {
		t.Fatalf("expected pane view lookup, got %q", got)
	}
}
