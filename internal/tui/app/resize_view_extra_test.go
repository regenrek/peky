package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
	"github.com/regenrek/peakypanes/internal/tui/views"
)

func buildResizeGeom(t *testing.T) resizeGeometry {
	t.Helper()
	preview := mouse.Rect{X: 0, Y: 0, W: 100, H: 40}
	rects := map[string]layout.Rect{
		"p1": {X: 0, Y: 0, W: 500, H: 500},
		"p2": {X: 500, Y: 0, W: 500, H: 500},
		"p3": {X: 0, Y: 500, W: 500, H: 500},
		"p4": {X: 500, Y: 500, W: 500, H: 500},
	}
	geom, ok := layoutgeom.Build(preview, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	return geom
}

func TestResizeViewHelpers(t *testing.T) {
	geom := buildResizeGeom(t)
	if len(geom.Edges) == 0 {
		t.Fatalf("expected edges")
	}
	edge := resizeEdgeRef{PaneID: geom.Edges[0].Ref.PaneID, Edge: geom.Edges[0].Ref.Edge}
	label, ok := resizeSizeLabel(geom, edge)
	if !ok || label == "" {
		t.Fatalf("expected size label")
	}

	guides := guidesForEdge(geom, edge, true)
	if len(guides) != 1 {
		t.Fatalf("expected guide")
	}
	cornerGuides := guidesForCorner(geom, geom.Corners[0].Ref, false)
	if len(cornerGuides) == 0 {
		t.Fatalf("expected corner guides")
	}
	clipped := adjustGuidesForPreviewLocal([]views.ResizeGuide{{X: -5, Y: -5, W: 10, H: 10, Active: true}}, geom.Preview)
	if len(clipped) == 0 {
		t.Fatalf("expected clipped guide")
	}

	m := newTestModelLite()
	m.resize.hover.hasEdge = true
	m.resize.hover.edge = edge
	g, _, _ := m.resizeOverlayGuides(geom)
	if len(g) == 0 {
		t.Fatalf("expected hover guides")
	}
	m.resize.drag.active = true
	m.resize.drag.edge = edge
	m.resize.drag.cursorSet = true
	m.resize.drag.cursorX = 5
	m.resize.drag.cursorY = 5
	layout := dashboardLayout{padLeft: 1, padTop: 1}
	posX, posY, ok := m.resizeLabelPosition(layout, geom, edge)
	if !ok || posX == 0 && posY == 0 {
		t.Fatalf("expected label position")
	}

	pane := m.selectedPane()
	if pane != nil {
		m.resize.drag.edge = resizeEdgeRef{PaneID: pane.ID, Edge: sessiond.ResizeEdgeRight}
		label := m.resizeEdgeLabel()
		if label == "" {
			t.Fatalf("expected edge label")
		}
	}
}
