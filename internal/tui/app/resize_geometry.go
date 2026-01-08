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
	sessionName, selectedID, preview, ok := m.resizeGeometryInputs()
	if !ok {
		return resizeGeometry{}, false
	}
	if geom, ok := m.resizeGeometryCached(sessionName, selectedID, preview); ok {
		return geom, true
	}
	geom, ok := m.buildResizeGeometry(sessionName, preview)
	if !ok {
		return resizeGeometry{}, false
	}
	m.storeResizeGeometryCache(sessionName, selectedID, preview, geom)
	return geom, true
}

func (m *Model) resizeGeometryInputs() (sessionName, selectedID string, preview mouse.Rect, ok bool) {
	if m == nil {
		return "", "", mouse.Rect{}, false
	}
	session := m.selectedSession()
	if session == nil || session.Name == "" {
		return "", "", mouse.Rect{}, false
	}
	selectedID = ""
	if sel := m.selectedPane(); sel != nil {
		selectedID = sel.ID
	}
	body, project, active, ok := m.projectPaneHitsContext()
	if !ok || project == nil || active == nil {
		return "", "", mouse.Rect{}, false
	}
	preview, ok = m.projectPaneHitsPreview(body, project)
	if !ok {
		return "", "", mouse.Rect{}, false
	}
	return session.Name, selectedID, preview, true
}

func (m *Model) resizeGeometryCached(sessionName, selectedID string, preview mouse.Rect) (resizeGeometry, bool) {
	if m == nil || !m.resize.cache.hasPreview {
		return resizeGeometry{}, false
	}
	cache := m.resize.cache
	if cache.session != sessionName || cache.version != m.layoutEngineVersion || cache.selected != selectedID || cache.preview != preview {
		return resizeGeometry{}, false
	}
	return cache.geometry, true
}

func (m *Model) storeResizeGeometryCache(sessionName, selectedID string, preview mouse.Rect, geom resizeGeometry) {
	if m == nil {
		return
	}
	m.resize.cache = resizeGeomCache{
		version:    m.layoutEngineVersion,
		session:    sessionName,
		selected:   selectedID,
		preview:    preview,
		geometry:   geom,
		hasPreview: true,
	}
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
