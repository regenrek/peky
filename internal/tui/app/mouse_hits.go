package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tui/layoutgeom"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
	"github.com/regenrek/peakypanes/internal/tui/panelayout"
)

type headerHitRect struct {
	Hit  mouse.HeaderHit
	Rect mouse.Rect
}

func (m *Model) hitTestPane(x, y int) (mouse.PaneHit, bool) {
	for _, hit := range m.paneHits() {
		if hit.Outer.Contains(x, y) {
			return hit, true
		}
	}
	return mouse.PaneHit{}, false
}

func (m *Model) hitTestHeader(x, y int) (mouse.HeaderHit, bool) {
	for _, hit := range m.headerHitRects() {
		if hit.Rect.Contains(x, y) {
			return hit.Hit, true
		}
	}
	return mouse.HeaderHit{}, false
}

func headerHitKind(kind headerPartKind) (mouse.HeaderKind, bool) {
	switch kind {
	case headerPartDashboard:
		return mouse.HeaderDashboard, true
	case headerPartProject:
		return mouse.HeaderProject, true
	case headerPartNew:
		return mouse.HeaderNew, true
	default:
		return mouse.HeaderDashboard, false
	}
}

func (m *Model) headerHitRects() []headerHitRect {
	header, ok := m.headerRect()
	if !ok {
		return nil
	}
	parts := m.headerParts()
	if len(parts) == 0 {
		return nil
	}

	lineHeight := 1
	if header.H < lineHeight {
		lineHeight = header.H
	}
	hits := make([]headerHitRect, 0, len(parts))
	cursor := header.X
	maxX := header.X + header.W
	for i, part := range parts {
		if i > 0 {
			cursor++
		}
		start := cursor
		end := cursor + part.Width
		if start >= maxX {
			break
		}
		visibleEnd := end
		if visibleEnd > maxX {
			visibleEnd = maxX
		}
		if part.Kind.clickable() && visibleEnd > start {
			kind, ok := headerHitKind(part.Kind)
			if !ok {
				cursor = end
				continue
			}
			hits = append(hits, headerHitRect{
				Hit: mouse.HeaderHit{
					Kind:      kind,
					ProjectID: part.ProjectID,
				},
				Rect: mouse.Rect{
					X: start,
					Y: header.Y,
					W: visibleEnd - start,
					H: lineHeight,
				},
			})
		}
		cursor = end
	}
	if label, hint, ok := m.updateBannerInfo(); ok {
		text := label
		if strings.TrimSpace(hint) != "" {
			text = text + " " + hint
		}
		textWidth := lipgloss.Width(text)
		if textWidth > 0 {
			start := header.X + header.W - textWidth
			if start < header.X {
				start = header.X
				textWidth = header.W
			}
			updateY := header.Y + 1
			if updateY < header.Y+header.H {
				hits = append(hits, headerHitRect{
					Hit: mouse.HeaderHit{Kind: mouse.HeaderUpdate},
					Rect: mouse.Rect{
						X: start,
						Y: updateY,
						W: textWidth,
						H: 1,
					},
				})
			}
		}
	}
	return hits
}

func (m *Model) paneHits() []mouse.PaneHit {
	var started time.Time
	if perfDebugEnabled() {
		started = time.Now()
	}
	if m.state != StateDashboard {
		m.logPaneViewSkipGlobal("state_not_dashboard", fmt.Sprintf("state=%d tab=%d", m.state, m.tab))
		if !started.IsZero() {
			logPerfEvery("tui.panehits.skip.state", 500*time.Millisecond,
				"tui: pane hits skip reason=state_not_dashboard state=%d tab=%d", m.state, m.tab)
		}
		return nil
	}
	if m.tab == TabProject {
		hits := m.projectPaneHits()
		if !started.IsZero() {
			logPerfEvery("tui.panehits.project", 500*time.Millisecond,
				"tui: pane hits computed scope=project count=%d dur=%s", len(hits), time.Since(started))
		}
		return hits
	}
	hits := m.dashboardPaneHits()
	if !started.IsZero() {
		logPerfEvery("tui.panehits.dashboard", 500*time.Millisecond,
			"tui: pane hits computed scope=dashboard count=%d dur=%s", len(hits), time.Since(started))
	}
	return hits
}

func (m *Model) projectPaneHits() []mouse.PaneHit {
	body, project, session, ok := m.projectPaneHitsContext()
	if !ok {
		return nil
	}
	if len(session.Panes) == 0 {
		m.logProjectPaneHitsSkip("no_panes", m.paneViewSkipContext(), "tui.panehits.project.empty")
		return nil
	}
	if perfDebugEnabled() {
		logPerfEvery("tui.panehits.project.meta", 500*time.Millisecond,
			"tui: pane hits project meta project=%s session=%s panes=%d",
			project.ID, session.Name, len(session.Panes))
	}
	preview, ok := m.projectPaneHitsPreview(body, project)
	if !ok {
		return nil
	}
	return m.projectPaneHitsForPreview(project, session, preview)
}

func (m *Model) projectPaneHitsContext() (mouse.Rect, *ProjectGroup, *SessionItem, bool) {
	body, ok := m.dashboardBodyRect()
	if !ok {
		m.logProjectPaneHitsSkip("body_rect_unavailable", m.paneViewSkipContext(), "tui.panehits.project.body")
		return mouse.Rect{}, nil, nil, false
	}
	project := m.selectedProject()
	session := m.selectedSession()
	if project == nil || session == nil {
		m.logProjectPaneHitsSkip("missing_selection", m.paneViewSkipContext(), "tui.panehits.project.selection")
		return mouse.Rect{}, nil, nil, false
	}
	return body, project, session, true
}

func (m *Model) projectPaneHitsPreview(body mouse.Rect, project *ProjectGroup) (mouse.Rect, bool) {
	if m.sidebarHidden(project) {
		preview := mouse.Rect{X: body.X, Y: body.Y, W: body.W, H: body.H}
		if !m.validateProjectPreviewRect(preview, "tui.panehits.project.preview.full") {
			return mouse.Rect{}, false
		}
		return preview, true
	}
	preview := m.projectSidebarPreviewRect(body)
	if !m.validateProjectPreviewRect(preview, "tui.panehits.project.preview") {
		return mouse.Rect{}, false
	}
	return preview, true
}

func (m *Model) projectSidebarPreviewRect(body mouse.Rect) mouse.Rect {
	base := body.W / 3
	leftWidth := clamp(base-(body.W/30), 22, 36)
	if leftWidth > body.W-10 {
		leftWidth = body.W / 2
	}
	rightWidth := body.W - leftWidth - 1
	if rightWidth < 10 {
		leftWidth = clamp(body.W/2, 12, body.W-10)
		rightWidth = body.W - leftWidth - 1
	}
	return mouse.Rect{
		X: body.X + leftWidth,
		Y: body.Y,
		W: rightWidth,
		H: body.H,
	}
}

func (m *Model) validateProjectPreviewRect(preview mouse.Rect, perfKey string) bool {
	if preview.W > 0 && preview.H > 0 {
		return true
	}
	m.logProjectPaneHitsSkip("preview_invalid", fmt.Sprintf("w=%d h=%d %s", preview.W, preview.H, m.paneViewSkipContext()), perfKey)
	return false
}

func (m *Model) projectPaneHitsForPreview(project *ProjectGroup, session *SessionItem, preview mouse.Rect) []mouse.PaneHit {
	rects := m.projectPaneLayoutRects(session)
	geom, ok := layoutgeom.Build(preview, rects)
	if !ok {
		return nil
	}
	return projectPaneLayoutHits(project, session, geom, m.settings.PaneTopbar.Enabled)
}

func (m *Model) logProjectPaneHitsSkip(reason, detail, perfKey string) {
	m.logPaneViewSkipGlobal(reason, detail)
	if !perfDebugEnabled() || perfKey == "" {
		return
	}
	logPerfEvery(perfKey, perfLogInterval, "tui: pane hits project skip reason=%s %s", reason, detail)
}

func (m *Model) projectPaneLayoutRects(session *SessionItem) map[string]layout.Rect {
	if session == nil {
		return nil
	}
	if engine := m.layoutEngineFor(session.Name); engine != nil && engine.Tree != nil {
		rects := engine.Tree.ViewRects()
		if len(rects) > 0 {
			return rects
		}
	}
	rects := make(map[string]layout.Rect, len(session.Panes))
	for _, pane := range session.Panes {
		if pane.ID == "" || pane.Width <= 0 || pane.Height <= 0 {
			continue
		}
		rects[pane.ID] = layout.Rect{
			X: pane.Left,
			Y: pane.Top,
			W: pane.Width,
			H: pane.Height,
		}
	}
	return rects
}

func projectPaneLayoutHits(project *ProjectGroup, session *SessionItem, geom layoutgeom.Geometry, paneTopbar bool) []mouse.PaneHit {
	if len(geom.Panes) == 0 {
		return nil
	}
	hits := make([]mouse.PaneHit, 0, len(geom.Panes))
	for _, pane := range geom.Panes {
		outer := pane.Screen
		if outer.Empty() {
			continue
		}
		content := layoutgeom.ContentRect(geom, outer)
		index := ""
		if session != nil {
			if item := findPaneByID(session.Panes, pane.ID); item != nil {
				index = item.Index
			}
		}
		topbar := mouse.Rect{}
		if paneTopbar && content.H >= 2 {
			topbar = mouse.Rect{X: content.X, Y: content.Y, W: content.W, H: 1}
			content = mouse.Rect{X: content.X, Y: content.Y + 1, W: content.W, H: content.H - 1}
		}
		hits = append(hits, mouse.PaneHit{
			PaneID: pane.ID,
			Selection: mouse.Selection{
				ProjectID: project.ID,
				Session:   session.Name,
				Pane:      index,
			},
			Outer:   outer,
			Topbar:  topbar,
			Content: content,
		})
	}
	return hits
}

func dashboardPaneContentRect(outer mouse.Rect, previewLines int) mouse.Rect {
	inner := tileInnerRect(outer, tileBorders{top: true, right: true, bottom: true, left: true})
	if inner.Empty() {
		return mouse.Rect{}
	}
	available := panelayout.DashboardTilePreviewLines(inner.H, previewLines)
	if available <= 0 {
		return mouse.Rect{}
	}
	return mouse.Rect{
		X: inner.X,
		Y: inner.Y + panelayout.DashboardTileHeaderLines,
		W: inner.W,
		H: available,
	}
}

func tileInnerRect(outer mouse.Rect, borders tileBorders) mouse.Rect {
	metrics := panelayout.TileMetricsFor(outer.W, outer.H, panelayout.TileBorders{
		Top:    borders.top,
		Left:   borders.left,
		Right:  borders.right,
		Bottom: borders.bottom,
	})
	return mouse.Rect{
		X: outer.X + metrics.ContentX,
		Y: outer.Y + metrics.ContentY,
		W: metrics.ContentWidth,
		H: metrics.InnerHeight,
	}
}
