package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestResizeEdgesForPane(t *testing.T) {
	m := newTestModelLite()
	session := findSessionByName(m.data.Projects, "alpha-1")
	if session == nil {
		t.Fatalf("expected session alpha-1")
		return
	}
	session.LayoutTree = &layout.TreeSnapshot{
		Root: layout.NodeSnapshot{
			Axis: layout.AxisHorizontal,
			Size: layout.LayoutBaseSize,
			Children: []layout.NodeSnapshot{
				{PaneID: "p1", Size: 500},
				{PaneID: "p2", Size: 500},
			},
		},
	}
	m.syncLayoutEngines()
	edges := m.resizeEdgesForPane("p1")
	found := false
	for _, edge := range edges {
		if edge.Edge == sessiond.ResizeEdgeRight {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected right edge for p1, got %#v", edges)
	}
}
