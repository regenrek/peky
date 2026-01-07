package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestPaneViewProvider(t *testing.T) {
	m := newTestModelLite()
	if m.paneViewProvider() != nil {
		t.Fatalf("expected nil pane view provider without client")
	}

	m.client = &sessiond.Client{}
	provider := m.paneViewProvider()
	if provider == nil {
		t.Fatalf("expected pane view provider with client")
	}
	key := paneViewKey{
		PaneID: "p1",
		Cols:   10,
		Rows:   5,
	}
	m.paneViews[key] = paneViewEntry{rendered: map[bool]string{false: "view"}}
	if got := provider("p1", 10, 5, false); got != "view" {
		t.Fatalf("expected pane view lookup, got %q", got)
	}
	if got := provider("", 10, 5, false); got != "" {
		t.Fatalf("expected empty result for missing id")
	}
}

func TestPreviewSessionForViewHidesNonZoomedPanes(t *testing.T) {
	m := newTestModelLite()
	session := findSessionByName(m.data.Projects, "alpha-1")
	if session == nil {
		t.Fatalf("expected session alpha-1")
	}
	if len(session.Panes) < 2 {
		t.Fatalf("expected at least 2 panes")
	}

	p1 := session.Panes[0].ID
	p2 := session.Panes[1].ID
	session.LayoutTree = &layout.TreeSnapshot{
		ZoomedPaneID: p1,
		Root: layout.NodeSnapshot{
			Axis: layout.AxisHorizontal,
			Size: layout.LayoutBaseSize,
			Children: []layout.NodeSnapshot{
				{PaneID: p1, Size: 500},
				{PaneID: p2, Size: 500},
			},
		},
	}
	m.syncLayoutEngines()

	preview := m.previewSessionForView(session)
	if preview == nil {
		t.Fatalf("expected preview session")
	}
	if len(preview.Panes) != len(session.Panes) {
		t.Fatalf("pane count mismatch")
	}

	var seenZoom bool
	var seenHidden bool
	for _, pane := range preview.Panes {
		switch pane.ID {
		case p1:
			seenZoom = true
			if pane.Width <= 0 || pane.Height <= 0 {
				t.Fatalf("zoomed pane should remain visible")
			}
		case p2:
			seenHidden = true
			if pane.Width != 0 || pane.Height != 0 {
				t.Fatalf("non-zoomed pane should be hidden, got w=%d h=%d", pane.Width, pane.Height)
			}
		}
	}
	if !seenZoom || !seenHidden {
		t.Fatalf("expected both zoomed and hidden panes to be observed")
	}
}
