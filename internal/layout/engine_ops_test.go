package layout

import (
	"fmt"
	"testing"
)

func newTwoPaneEngine(t *testing.T) *Engine {
	t.Helper()
	tree, err := BuildTree(&LayoutConfig{Grid: "1x2"}, []string{"p1", "p2"})
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}
	return NewEngine(tree)
}

func TestBuildTreeGrid(t *testing.T) {
	tree, err := BuildTree(&LayoutConfig{Grid: "1x2"}, []string{"p1", "p2"})
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}
	if tree.Root == nil || tree.Root.Axis != AxisHorizontal || len(tree.Root.Children) != 2 {
		t.Fatalf("unexpected tree root: %#v", tree.Root)
	}
	rects := tree.Rects()
	left := rects["p1"]
	right := rects["p2"]
	if left.W != LayoutBaseSize/2 || right.W != LayoutBaseSize/2 {
		t.Fatalf("unexpected widths: %#v %#v", left, right)
	}
	if left.X != 0 || right.X != left.W {
		t.Fatalf("unexpected positions: %#v %#v", left, right)
	}
}

func TestEngineResizeAndReset(t *testing.T) {
	engine := newTwoPaneEngine(t)
	result, err := engine.Apply(ResizeOp{PaneID: "p1", Edge: ResizeEdgeRight, Delta: 100, Snap: false})
	if err != nil {
		t.Fatalf("Apply(resize) error: %v", err)
	}
	if !result.Changed || result.Snapped {
		t.Fatalf("unexpected resize result: %#v", result)
	}
	rects := engine.Tree.Rects()
	if rects["p1"].W != 600 || rects["p2"].W != 400 {
		t.Fatalf("unexpected resize rects: %#v", rects)
	}

	_, err = engine.Apply(ResetSizesOp{})
	if err != nil {
		t.Fatalf("Apply(reset) error: %v", err)
	}
	rects = engine.Tree.Rects()
	if rects["p1"].W != LayoutBaseSize/2 || rects["p2"].W != LayoutBaseSize/2 {
		t.Fatalf("unexpected reset rects: %#v", rects)
	}
	if len(engine.History.Past) != 0 {
		t.Fatalf("expected history cleared on reset")
	}
}

func TestEngineSwap(t *testing.T) {
	engine := newTwoPaneEngine(t)
	before := engine.Tree.Rects()
	if _, err := engine.Apply(SwapOp{PaneA: "p1", PaneB: "p2"}); err != nil {
		t.Fatalf("Apply(swap) error: %v", err)
	}
	after := engine.Tree.Rects()
	if after["p1"] != before["p2"] || after["p2"] != before["p1"] {
		t.Fatalf("swap did not exchange rects")
	}
}

func TestEngineZoomViewRects(t *testing.T) {
	engine := newTwoPaneEngine(t)
	result, err := engine.Apply(ZoomOp{PaneID: "p1", Toggle: true})
	if err != nil {
		t.Fatalf("Apply(zoom) error: %v", err)
	}
	if len(result.Affected) != 2 {
		t.Fatalf("expected affected panes on zoom, got %#v", result.Affected)
	}
	if engine.Tree.ZoomedPaneID != "p1" {
		t.Fatalf("expected zoomed pane p1, got %q", engine.Tree.ZoomedPaneID)
	}
	view := engine.Tree.ViewRects()
	if len(view) != 1 || view["p1"].W != LayoutBaseSize || view["p1"].H != LayoutBaseSize {
		t.Fatalf("unexpected view rects: %#v", view)
	}
	if _, err := engine.Apply(ZoomOp{PaneID: "p1", Toggle: true}); err != nil {
		t.Fatalf("Apply(zoom toggle) error: %v", err)
	}
	if engine.Tree.ZoomedPaneID != "" {
		t.Fatalf("expected zoom cleared")
	}
}

func TestSnapPosition(t *testing.T) {
	cfg := DefaultSnapConfig()
	pos, state := SnapPosition(cfg, 495, 100, 900, SnapState{})
	if !state.Active || pos != 500 {
		t.Fatalf("expected snap to 500, got pos=%d state=%#v", pos, state)
	}
	pos, state = SnapPosition(cfg, 506, 100, 900, state)
	if pos != 500 || !state.Active {
		t.Fatalf("expected hysteresis to hold snap, got pos=%d state=%#v", pos, state)
	}
}

func TestEngineResizeMultiChildEdge(t *testing.T) {
	tree, err := BuildTree(&LayoutConfig{Grid: "1x3"}, []string{"p1", "p2", "p3"})
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}
	engine := NewEngine(tree)
	if _, err := engine.Apply(ResizeOp{PaneID: "p2", Edge: ResizeEdgeLeft, Delta: 25, Snap: false}); err != nil {
		t.Fatalf("Apply(resize multi-child) error: %v", err)
	}
}

func TestSnapAlignsToNeighborEdge(t *testing.T) {
	tree, err := BuildTree(&LayoutConfig{Grid: "1x3"}, []string{"p1", "p2", "p3"})
	if err != nil {
		t.Fatalf("BuildTree() error: %v", err)
	}
	engine := NewEngine(tree)
	engine.Snap = SnapConfig{Threshold: 10, Hysteresis: 3}
	rects := engine.Tree.Rects()
	want := rects["p1"].W

	result, err := engine.Apply(ResizeOp{PaneID: "p1", Edge: ResizeEdgeRight, Delta: -5, Snap: true})
	if err != nil {
		t.Fatalf("Apply(resize snap) error: %v", err)
	}
	if !result.Snapped || !result.SnapState.Active {
		t.Fatalf("expected snap active, got %#v", result)
	}
	if result.SnapState.Target != want {
		t.Fatalf("expected snap target %d, got %d", want, result.SnapState.Target)
	}
}

func TestHistoryUndoRedo(t *testing.T) {
	engine := newTwoPaneEngine(t)
	if _, err := engine.Apply(ResizeOp{PaneID: "p1", Edge: ResizeEdgeRight, Delta: 50}); err != nil {
		t.Fatalf("Apply(resize) error: %v", err)
	}
	current := engine.Tree.Clone()
	undo, ok := engine.History.Undo(current)
	if !ok || undo == nil {
		t.Fatalf("expected undo snapshot")
	}
	redo, ok := engine.History.Redo(undo)
	if !ok || redo == nil {
		t.Fatalf("expected redo snapshot")
	}
}

func BenchmarkResizeGrid(b *testing.B) {
	paneIDs := buildPaneIDs(12)
	tree, err := BuildTree(&LayoutConfig{Grid: "3x4"}, paneIDs)
	if err != nil {
		b.Fatalf("BuildTree error: %v", err)
	}
	engine := NewEngine(tree)
	engine.Constraints = Constraints{MinWidth: 1, MinHeight: 1}
	op := ResizeOp{PaneID: paneIDs[0], Edge: ResizeEdgeRight, Delta: 5, Snap: false}
	undo := ResizeOp{PaneID: paneIDs[0], Edge: ResizeEdgeRight, Delta: -5, Snap: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Apply(op); err != nil {
			b.Fatalf("Apply(resize) error: %v", err)
		}
		if _, err := engine.Apply(undo); err != nil {
			b.Fatalf("Apply(resize undo) error: %v", err)
		}
	}
}

func BenchmarkResizeDeepTree(b *testing.B) {
	paneIDs := buildPaneIDs(10)
	defs := make([]PaneDef, len(paneIDs))
	for i := 1; i < len(defs); i++ {
		defs[i].Split = "horizontal"
	}
	tree, err := BuildTree(&LayoutConfig{Panes: defs}, paneIDs)
	if err != nil {
		b.Fatalf("BuildTree error: %v", err)
	}
	engine := NewEngine(tree)
	engine.Constraints = Constraints{MinWidth: 1, MinHeight: 1}
	op := ResizeOp{PaneID: paneIDs[0], Edge: ResizeEdgeRight, Delta: 5, Snap: false}
	undo := ResizeOp{PaneID: paneIDs[0], Edge: ResizeEdgeRight, Delta: -5, Snap: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Apply(op); err != nil {
			b.Fatalf("Apply(resize) error: %v", err)
		}
		if _, err := engine.Apply(undo); err != nil {
			b.Fatalf("Apply(resize undo) error: %v", err)
		}
	}
}

func buildPaneIDs(count int) []string {
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = fmt.Sprintf("p-%d", i+1)
	}
	return ids
}
