package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestResizeNudgeForKey(t *testing.T) {
	step, axis, ok := resizeNudgeForKey("left")
	if !ok || step != resizeNudgeStep || axis != layout.AxisHorizontal {
		t.Fatalf("unexpected nudge for left: step=%d axis=%v ok=%v", step, axis, ok)
	}
	step, axis, ok = resizeNudgeForKey("shift+up")
	if !ok || step != resizeNudgeStepFast || axis != layout.AxisVertical {
		t.Fatalf("unexpected nudge for shift+up")
	}
	if _, _, ok := resizeNudgeForKey("unknown"); ok {
		t.Fatalf("expected unknown nudge to be false")
	}
}

func TestResizeEdgeMatchesAxis(t *testing.T) {
	if !resizeEdgeMatchesAxis(sessiond.ResizeEdgeLeft, layout.AxisHorizontal) {
		t.Fatalf("expected left edge to match horizontal")
	}
	if resizeEdgeMatchesAxis(sessiond.ResizeEdgeLeft, layout.AxisVertical) {
		t.Fatalf("expected left edge to not match vertical")
	}
}

func TestResizeDeltaForAxis(t *testing.T) {
	if delta, ok := resizeDeltaForAxis(layout.AxisHorizontal, "left", 10); !ok || delta != -10 {
		t.Fatalf("unexpected delta for left: %d ok=%v", delta, ok)
	}
	if delta, ok := resizeDeltaForAxis(layout.AxisVertical, "down", 5); !ok || delta != 5 {
		t.Fatalf("unexpected delta for down: %d ok=%v", delta, ok)
	}
	if _, ok := resizeDeltaForAxis(layout.AxisVertical, "left", 5); ok {
		t.Fatalf("expected invalid delta for mismatched axis")
	}
}

func TestUniqueEdges(t *testing.T) {
	edges := uniqueEdges([]resizeEdgeRef{
		{PaneID: "p1", Edge: sessiond.ResizeEdgeLeft},
		{PaneID: "p1", Edge: sessiond.ResizeEdgeLeft},
		{PaneID: "p1", Edge: sessiond.ResizeEdgeRight},
	})
	if len(edges) != 2 {
		t.Fatalf("expected unique edges, got %#v", edges)
	}
}

func TestRangesOverlap(t *testing.T) {
	if !rangesOverlap(0, 10, 5, 15) {
		t.Fatalf("expected ranges to overlap")
	}
	if rangesOverlap(0, 5, 6, 10) {
		t.Fatalf("expected ranges to not overlap")
	}
}
