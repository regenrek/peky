package app

import (
	"fmt"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

type resizeApplyResult struct {
	snapped   bool
	snapState sessiond.SnapState
}

func cloneLayoutEngine(engine *layout.Engine) *layout.Engine {
	if engine == nil || engine.Tree == nil {
		return nil
	}
	clone := layout.NewEngine(engine.Tree.Clone())
	clone.Constraints = engine.Constraints
	clone.Snap = engine.Snap
	return clone
}

func applyResizeToEngine(engine *layout.Engine, edge resizeEdgeRef, delta int, snap bool, snapState sessiond.SnapState) (resizeApplyResult, error) {
	if engine == nil {
		return resizeApplyResult{}, fmt.Errorf("resize preview: engine missing")
	}
	layoutEdge, ok := layoutEdgeFromSession(edge.Edge)
	if !ok {
		return resizeApplyResult{}, fmt.Errorf("resize preview: invalid edge %q", edge.Edge)
	}
	result, err := engine.Apply(layout.ResizeOp{
		PaneID:    edge.PaneID,
		Edge:      layoutEdge,
		Delta:     delta,
		Snap:      snap,
		SnapState: snapStateToLayout(snapState),
	})
	if err != nil {
		return resizeApplyResult{}, err
	}
	return resizeApplyResult{
		snapped:   result.Snapped,
		snapState: snapStateFromLayout(result.SnapState),
	}, nil
}

func layoutEdgeFromSession(edge sessiond.ResizeEdge) (layout.ResizeEdge, bool) {
	switch edge {
	case sessiond.ResizeEdgeLeft:
		return layout.ResizeEdgeLeft, true
	case sessiond.ResizeEdgeRight:
		return layout.ResizeEdgeRight, true
	case sessiond.ResizeEdgeUp:
		return layout.ResizeEdgeUp, true
	case sessiond.ResizeEdgeDown:
		return layout.ResizeEdgeDown, true
	default:
		return layout.ResizeEdgeLeft, false
	}
}

func snapStateToLayout(state sessiond.SnapState) layout.SnapState {
	return layout.SnapState{Active: state.Active, Target: state.Target}
}

func snapStateFromLayout(state layout.SnapState) sessiond.SnapState {
	return sessiond.SnapState{Active: state.Active, Target: state.Target}
}
