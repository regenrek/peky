package app

import (
	"fmt"
	"math"
	"strings"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
	"github.com/regenrek/peakypanes/internal/tui/views"
)

func (m *Model) resizeOverlayView() views.ResizeOverlay {
	if m == nil || m.state != StateDashboard || m.tab != TabProject {
		return views.ResizeOverlay{}
	}
	geom, ok := m.resizeGeometry()
	if !ok {
		return views.ResizeOverlay{}
	}
	layout, ok := m.dashboardLayoutInternal("resizeOverlay")
	if !ok {
		return views.ResizeOverlay{}
	}
	snapEnabled := m.resize.snap
	snapActive := m.resize.key.snapState.Active
	if m.resize.drag.active {
		snapEnabled = m.resize.drag.snapEnabled
		snapActive = m.resize.drag.snapState.Active || m.resize.drag.snapStateAlt.Active
	}
	overlay := views.ResizeOverlay{
		Active:      m.resize.mode,
		Dragging:    m.resize.drag.active,
		SnapEnabled: snapEnabled,
		SnapActive:  snapActive,
	}
	if m.keys != nil {
		overlay.ModeKey = keyLabel(m.keys.resizeMode)
	}

	var guides []views.ResizeGuide
	var activeEdge resizeEdgeRef
	activeEdgeOK := false
	switch {
	case m.resize.drag.active:
		activeEdge = m.resize.drag.edge
		activeEdgeOK = activeEdge.PaneID != ""
		if m.resize.drag.cornerActive {
			activeEdge = m.resize.drag.corner.Vertical
			activeEdgeOK = activeEdge.PaneID != ""
		}
		guides = append(guides, m.resizeGuidesForActive(geom, true)...)
	case m.resize.mode:
		activeEdge, activeEdgeOK = m.activeResizeEdge()
		guides = append(guides, m.resizeGuidesForActive(geom, false)...)
	case m.resize.hover.hasEdge || m.resize.hover.hasCorner:
		guides = append(guides, m.resizeGuidesForHover(geom)...)
	}

	if len(guides) > 0 {
		overlay.Guides = adjustGuidesForPreviewLocal(guides, geom.Preview)
	}
	overlay.EdgeLabel = m.resizeEdgeLabel()
	if m.resize.drag.active && activeEdgeOK {
		if label, ok := resizeSizeLabel(geom, activeEdge); ok {
			if x, y, ok := m.resizeLabelPosition(layout, geom, activeEdge); ok {
				overlay.Label = label
				overlay.LabelX = x
				overlay.LabelY = y
			}
		}
	}
	return overlay
}

func (m *Model) resizeGuidesForActive(geom resizeGeometry, dragging bool) []views.ResizeGuide {
	if m.resize.drag.active {
		if m.resize.drag.cornerActive {
			return guidesForCorner(geom, m.resize.drag.corner, dragging)
		}
		return guidesForEdge(geom, m.resize.drag.edge, dragging)
	}
	edge, ok := m.activeResizeEdge()
	if !ok {
		return nil
	}
	return guidesForEdge(geom, edge, dragging)
}

func (m *Model) resizeGuidesForHover(geom resizeGeometry) []views.ResizeGuide {
	if m.resize.hover.hasCorner {
		return guidesForCorner(geom, m.resize.hover.corner, false)
	}
	if m.resize.hover.hasEdge {
		return guidesForEdge(geom, m.resize.hover.edge, false)
	}
	return nil
}

func guidesForEdge(geom resizeGeometry, edge resizeEdgeRef, active bool) []views.ResizeGuide {
	rect, ok := edgeScreenRectForRef(geom, edge)
	if !ok {
		return nil
	}
	return []views.ResizeGuide{toViewGuide(rect, active)}
}

func guidesForCorner(geom resizeGeometry, corner resizeCornerRef, active bool) []views.ResizeGuide {
	guides := make([]views.ResizeGuide, 0, 2)
	if rect, ok := edgeScreenRectForRef(geom, corner.Vertical); ok {
		guides = append(guides, toViewGuide(rect, active))
	}
	if rect, ok := edgeScreenRectForRef(geom, corner.Horizontal); ok {
		guides = append(guides, toViewGuide(rect, active))
	}
	return guides
}

func adjustGuidesForPreviewLocal(guides []views.ResizeGuide, preview mouse.Rect) []views.ResizeGuide {
	if len(guides) == 0 || preview.Empty() {
		return nil
	}
	out := make([]views.ResizeGuide, 0, len(guides))
	for _, guide := range guides {
		guide.X -= preview.X
		guide.Y -= preview.Y
		if guide.W <= 0 || guide.H <= 0 {
			continue
		}
		x0 := maxInt(guide.X, 0)
		y0 := maxInt(guide.Y, 0)
		x1 := minInt(guide.X+guide.W, preview.W)
		y1 := minInt(guide.Y+guide.H, preview.H)
		if x1 <= x0 || y1 <= y0 {
			continue
		}
		guide.X = x0
		guide.Y = y0
		guide.W = x1 - x0
		guide.H = y1 - y0
		out = append(out, guide)
	}
	return out
}

func toViewGuide(rect mouse.Rect, active bool) views.ResizeGuide {
	return views.ResizeGuide{X: rect.X, Y: rect.Y, W: rect.W, H: rect.H, Active: active}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func resizeSizeLabel(geom resizeGeometry, edge resizeEdgeRef) (string, bool) {
	pane, neighbor, ok := panesForEdge(geom, edge)
	if !ok {
		return "", false
	}
	total := 0
	size := 0
	switch edge.Edge {
	case sessiond.ResizeEdgeLeft, sessiond.ResizeEdgeRight:
		total = pane.Layout.W + neighbor.Layout.W
		size = pane.Layout.W
	case sessiond.ResizeEdgeUp, sessiond.ResizeEdgeDown:
		total = pane.Layout.H + neighbor.Layout.H
		size = pane.Layout.H
	default:
		return "", false
	}
	if total <= 0 {
		return "", false
	}
	percent := int(math.Round(float64(size) / float64(total) * 100))
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	cols := pane.Screen.W
	rows := pane.Screen.H
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	return fmt.Sprintf("%d%% • %dx%d", percent, cols, rows), true
}

func (m *Model) resizeLabelPosition(layout dashboardLayout, geom resizeGeometry, edge resizeEdgeRef) (int, int, bool) {
	if m == nil {
		return 0, 0, false
	}
	if m.resize.drag.cursorSet {
		return m.resize.drag.cursorX - layout.padLeft + 1, m.resize.drag.cursorY - layout.padTop + 1, true
	}
	rect, ok := edgeScreenRectForRef(geom, edge)
	if !ok {
		return 0, 0, false
	}
	x := rect.X + rect.W/2 - layout.padLeft
	y := rect.Y + rect.H/2 - layout.padTop
	return x, y, true
}

func (m *Model) resizeEdgeLabel() string {
	edge, ok := m.activeResizeEdge()
	if m.resize.drag.active {
		edge = m.resize.drag.edge
		ok = edge.PaneID != ""
		if m.resize.drag.cornerActive {
			edge = m.resize.drag.corner.Vertical
			ok = edge.PaneID != ""
		}
	}
	if !ok {
		return ""
	}
	pane := m.paneByID(edge.PaneID)
	if pane == nil {
		return ""
	}
	name := strings.TrimSpace(pane.Title)
	if name == "" {
		name = "pane " + pane.Index
	}
	return name + " • " + strings.ToLower(string(edge.Edge))
}
