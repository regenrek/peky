package layoutgeom

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestEdgeRectsAndNeighbors(t *testing.T) {
	preview := mouse.Rect{X: 0, Y: 0, W: 100, H: 40}
	rects := map[string]layout.Rect{
		"left":  {X: 0, Y: 0, W: 500, H: layout.LayoutBaseSize},
		"right": {X: 500, Y: 0, W: 500, H: layout.LayoutBaseSize},
	}
	geom, ok := Build(preview, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	ref := EdgeRef{PaneID: "left", Edge: sessiond.ResizeEdgeRight}
	line, ok := EdgeLineRect(geom, ref)
	if !ok || line.Empty() {
		t.Fatalf("expected edge line rect")
	}
	hit, ok := EdgeHitRect(geom, ref)
	if !ok || hit.Empty() {
		t.Fatalf("expected edge hit rect")
	}
	left, right, ok := PanesForEdge(geom, ref)
	if !ok || left.ID != "left" || right.ID != "right" {
		t.Fatalf("unexpected panes for edge: %#v %#v ok=%v", left, right, ok)
	}
}

func TestEdgeRectFallbackFromPane(t *testing.T) {
	preview := mouse.Rect{X: 0, Y: 0, W: 80, H: 20}
	rects := map[string]layout.Rect{
		"solo": {X: 0, Y: 0, W: layout.LayoutBaseSize, H: layout.LayoutBaseSize},
	}
	geom, ok := Build(preview, rects)
	if !ok {
		t.Fatalf("expected geometry")
	}
	ref := EdgeRef{PaneID: "solo", Edge: sessiond.ResizeEdgeRight}
	line, ok := EdgeLineRect(geom, ref)
	if !ok || line.Empty() {
		t.Fatalf("expected edge line rect from pane")
	}
	hit, ok := EdgeHitRect(geom, ref)
	if !ok || hit.Empty() {
		t.Fatalf("expected edge hit rect from pane")
	}
}

func TestLayoutPosFromScreen(t *testing.T) {
	preview := mouse.Rect{X: 10, Y: 5, W: 100, H: 50}
	x, y, ok := LayoutPosFromScreen(preview, 5, 2)
	if !ok {
		t.Fatalf("expected layout position")
	}
	if x < 0 || y < 0 {
		t.Fatalf("expected non-negative layout pos")
	}
	x, y, ok = LayoutPosFromScreen(preview, 999, 999)
	if !ok {
		t.Fatalf("expected clamped layout position")
	}
	if x > layout.LayoutBaseSize || y > layout.LayoutBaseSize {
		t.Fatalf("expected clamped layout pos within base size")
	}
}
