package app

import (
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

type resizeGeometry = layoutgeom.Geometry

type resizePane = layoutgeom.Pane

type resizeHitKind int

const (
	resizeHitNone resizeHitKind = iota
	resizeHitCorner
	resizeHitEdge
	resizeHitPane
)

type resizeHit struct {
	Kind   resizeHitKind
	Edge   resizeEdgeRef
	Corner resizeCornerRef
	PaneID string
}

func (m *Model) resizeGeometry() (resizeGeometry, bool) {
	if m == nil || m.state != StateDashboard || m.tab != TabProject {
		return resizeGeometry{}, false
	}
	session := m.selectedSession()
	if session == nil || session.Name == "" {
		return resizeGeometry{}, false
	}
	selectedID := ""
	if sel := m.selectedPane(); sel != nil {
		selectedID = sel.ID
	}
	body, project, active, ok := m.projectPaneHitsContext()
	if !ok || project == nil || active == nil {
		return resizeGeometry{}, false
	}
	preview, ok := m.projectPaneHitsPreview(body, project)
	if !ok {
		return resizeGeometry{}, false
	}
	if m.resize.cache.hasPreview &&
		m.resize.cache.session == session.Name &&
		m.resize.cache.version == m.layoutEngineVersion &&
		m.resize.cache.selected == selectedID &&
		m.resize.cache.preview == preview {
		return m.resize.cache.geometry, true
	}
	geom, ok := m.buildResizeGeometry(session.Name, preview)
	if !ok {
		return resizeGeometry{}, false
	}
	m.resize.cache = resizeGeomCache{
		version:    m.layoutEngineVersion,
		session:    session.Name,
		selected:   selectedID,
		preview:    preview,
		geometry:   geom,
		hasPreview: true,
	}
	return geom, true
}

func (m *Model) buildResizeGeometry(sessionName string, preview mouse.Rect) (resizeGeometry, bool) {
	engine := m.layoutEngineFor(sessionName)
	if engine == nil || engine.Tree == nil {
		return resizeGeometry{}, false
	}
	rects := engine.Tree.ViewRects()
	if len(rects) == 0 {
		return resizeGeometry{}, false
	}
	return layoutgeom.Build(preview, rects)
}

func (m *Model) resizeHitTest(x, y int) (resizeHit, bool) {
	geom, ok := m.resizeGeometry()
	if !ok {
		return resizeHit{}, false
	}
	for _, corner := range geom.Corners {
		if corner.HitRect.Contains(x, y) {
			return resizeHit{Kind: resizeHitCorner, Corner: corner.Ref}, true
		}
	}
	for _, edge := range geom.Edges {
		if edge.HitRect.Contains(x, y) {
			return resizeHit{Kind: resizeHitEdge, Edge: edge.Ref}, true
		}
	}
	for _, pane := range geom.Panes {
		if pane.Screen.Contains(x, y) {
			return resizeHit{Kind: resizeHitPane, PaneID: pane.ID}, true
		}
	}
	return resizeHit{}, false
}

func edgeScreenRectForRef(geom resizeGeometry, ref resizeEdgeRef) (mouse.Rect, bool) {
	return layoutgeom.EdgeLineRect(geom, ref)
}

func panesForEdge(geom resizeGeometry, ref resizeEdgeRef) (resizePane, resizePane, bool) {
	return layoutgeom.PanesForEdge(geom, ref)
}
