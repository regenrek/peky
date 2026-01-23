package layout

import "testing"

func TestEngineApplyErrors(t *testing.T) {
	var nilEngine *Engine
	if _, err := nilEngine.Apply(ResizeOp{}); err == nil {
		t.Fatalf("expected error for nil engine")
	}
	engine := &Engine{}
	if _, err := engine.Apply(ResizeOp{}); err == nil {
		t.Fatalf("expected error for nil tree")
	}
}

type bogusOp struct{}

func (bogusOp) Kind() OpKind { return OpKind("bogus") }

func TestEngineApplyUnknownOp(t *testing.T) {
	engine := newTwoPaneEngine(t)
	if _, err := engine.Apply(bogusOp{}); err == nil {
		t.Fatalf("expected error for unknown op")
	}
}

func TestApplyCloseSinglePane(t *testing.T) {
	tree, err := BuildTree(&LayoutConfig{Grid: "1x1"}, []string{"p1"})
	if err != nil {
		t.Fatalf("BuildTree error: %v", err)
	}
	engine := NewEngine(tree)
	result, err := engine.Apply(CloseOp{PaneID: "p1"})
	if err != nil {
		t.Fatalf("Apply close error: %v", err)
	}
	if !result.Changed || engine.Tree.Root != nil {
		t.Fatalf("expected root cleared, result=%#v", result)
	}
}

func TestApplySwapErrors(t *testing.T) {
	engine := newTwoPaneEngine(t)
	if _, err := engine.Apply(SwapOp{PaneA: "", PaneB: "p2"}); err == nil {
		t.Fatalf("expected error for empty pane id")
	}
	if _, err := engine.Apply(SwapOp{PaneA: "p1", PaneB: "p1"}); err != nil {
		t.Fatalf("unexpected error for same pane: %v", err)
	}
	if _, err := engine.Apply(SwapOp{PaneA: "p1", PaneB: "missing"}); err == nil {
		t.Fatalf("expected error for missing pane")
	}
}

func TestApplyResetSizesErrors(t *testing.T) {
	engine := newTwoPaneEngine(t)
	if _, err := engine.Apply(ResetSizesOp{PaneID: "missing"}); err == nil {
		t.Fatalf("expected error for missing pane")
	}
	engine.Tree.Root = nil
	if _, err := engine.Apply(ResetSizesOp{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSplitComputeSizesErrors(t *testing.T) {
	if _, _, err := splitComputeSizes(1, 50, 1); err == nil {
		t.Fatalf("expected error for small total")
	}
	if _, _, err := splitComputeSizes(10, 50, 6); err == nil {
		t.Fatalf("expected min size error")
	}
}

func TestReplaceLeafWithSplit(t *testing.T) {
	if err := replaceLeafWithSplit(nil, nil, nil, nil); err == nil {
		t.Fatalf("expected error for nil tree")
	}
	tree, err := BuildTree(&LayoutConfig{Grid: "1x1"}, []string{"p1"})
	if err != nil {
		t.Fatalf("BuildTree error: %v", err)
	}
	root := tree.Root
	split := &Node{ID: tree.nextNodeID()}
	if err := replaceLeafWithSplit(tree, nil, root, split); err != nil {
		t.Fatalf("replace error: %v", err)
	}
	if tree.Root != split {
		t.Fatalf("expected split as root")
	}
}

func TestFindSplitForEdgeNoMatch(t *testing.T) {
	tree, err := BuildTree(&LayoutConfig{Grid: "1x1"}, []string{"p1"})
	if err != nil {
		t.Fatalf("BuildTree error: %v", err)
	}
	leaf := tree.Leaf("p1")
	if _, _, _, err := findSplitForEdge(leaf, ResizeEdgeRight); err == nil {
		t.Fatalf("expected no split error")
	}
	if _, _, _, err := findSplitForEdge(nil, ResizeEdgeRight); err == nil {
		t.Fatalf("expected error for nil leaf")
	}
}
