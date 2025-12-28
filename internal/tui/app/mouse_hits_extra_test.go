package app

import (
	"testing"

	"github.com/muesli/termenv"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestDashboardPaneHits(t *testing.T) {
	m := newTestModelLite()
	m.state = StateDashboard
	m.tab = TabDashboard
	m.selection = selectionState{Project: "Alpha", Session: "alpha-1", Pane: "1"}

	hits := m.dashboardPaneHits()
	if len(hits) == 0 {
		t.Fatalf("expected dashboard hits")
	}
	if hits[0].Selection.Project == "" || hits[0].Selection.Session == "" {
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

func TestProjectPaneHitsLayoutAndGrid(t *testing.T) {
	m := newTestModelLite()
	m.state = StateDashboard
	m.tab = TabProject
	m.selection = selectionState{Project: "Alpha", Session: "alpha-1", Pane: "1"}

	m.settings.PreviewMode = "layout"
	layoutHits := m.projectPaneHits()
	if len(layoutHits) != len(m.selectedSession().Panes) {
		t.Fatalf("expected layout hits for panes")
	}

	m.settings.PreviewMode = "grid"
	gridHits := m.projectPaneHits()
	if len(gridHits) == 0 {
		t.Fatalf("expected grid hits")
	}
	if gridHits[0].Content.Empty() {
		t.Fatalf("expected grid content rect")
	}
}

func TestPaneViewRequestsAndLookup(t *testing.T) {
	m := newTestModelLite()
	m.state = StateDashboard
	m.tab = TabProject
	m.selection = selectionState{Project: "Alpha", Session: "alpha-1", Pane: "1"}

	reqs := m.paneViewRequests()
	if len(reqs) == 0 {
		t.Fatalf("expected pane view requests")
	}
	hit, ok := m.paneHitFor("p1")
	if !ok {
		t.Fatalf("expected pane hit for p1")
	}
	req := m.paneViewRequestForHit(hit)
	if req == nil || req.PaneID != "p1" {
		t.Fatalf("expected pane view request, got %#v", req)
	}

	key := paneViewKey{
		PaneID:       "p1",
		Cols:         req.Cols,
		Rows:         req.Rows,
		Mode:         sessiond.PaneViewANSI,
		ShowCursor:   false,
		ColorProfile: termenv.TrueColor,
	}
	m.paneViews[key] = "view"
	if got := m.paneView("p1", req.Cols, req.Rows, sessiond.PaneViewANSI, false, termenv.TrueColor); got != "view" {
		t.Fatalf("expected pane view lookup, got %q", got)
	}
}
