package app

import (
	"strings"

	"github.com/regenrek/peakypanes/internal/tui/mouse"
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
					Kind:        kind,
					ProjectName: part.ProjectName,
				},
				Rect: mouse.Rect{
					X: start,
					Y: header.Y,
					W: visibleEnd - start,
					H: header.H,
				},
			})
		}
		cursor = end
	}
	return hits
}

func (m *Model) paneHits() []mouse.PaneHit {
	if m.state != StateDashboard {
		return nil
	}
	if m.tab == TabProject {
		return m.projectPaneHits()
	}
	return m.dashboardPaneHits()
}

func (m *Model) projectPaneHits() []mouse.PaneHit {
	body, ok := m.dashboardBodyRect()
	if !ok {
		return nil
	}
	project := m.selectedProject()
	session := m.selectedSession()
	if project == nil || session == nil {
		return nil
	}
	if len(session.Panes) == 0 {
		return nil
	}

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

	preview := mouse.Rect{
		X: body.X + leftWidth,
		Y: body.Y,
		W: rightWidth,
		H: body.H,
	}

	if preview.W <= 0 || preview.H <= 0 {
		return nil
	}

	mode := m.settings.PreviewMode
	if mode == "layout" {
		return projectPaneLayoutHits(project, session, session.Panes, preview)
	}
	return projectPaneTileHits(project, session, session.Panes, preview)
}

func projectPaneLayoutHits(project *ProjectGroup, session *SessionItem, panes []PaneItem, preview mouse.Rect) []mouse.PaneHit {
	maxW, maxH := paneBounds(panes)
	if maxW == 0 || maxH == 0 {
		return nil
	}

	hits := make([]mouse.PaneHit, 0, len(panes))
	for _, pane := range panes {
		x1, y1, w, h := scalePane(pane, maxW, maxH, preview.W, preview.H)
		if w <= 0 || h <= 0 {
			continue
		}
		outer := mouse.Rect{
			X: preview.X + x1,
			Y: preview.Y + y1,
			W: w,
			H: h,
		}
		hits = append(hits, mouse.PaneHit{
			PaneID: pane.ID,
			Selection: mouse.Selection{
				Project: project.Name,
				Session: session.Name,
				Pane:    pane.Index,
			},
			Outer:   outer,
			Content: mouse.Rect{},
		})
	}
	return hits
}

func projectPaneTileHits(project *ProjectGroup, session *SessionItem, panes []PaneItem, preview mouse.Rect) []mouse.PaneHit {
	cols := 3
	if preview.W < 70 {
		cols = 2
	}
	if preview.W < 42 {
		cols = 1
	}
	if len(panes) < cols {
		cols = len(panes)
	}
	if cols <= 0 {
		cols = 1
	}

	rows := (len(panes) + cols - 1) / cols
	gap := 0
	availableHeight := preview.H - gap*(rows-1)
	if availableHeight < rows {
		availableHeight = rows
	}
	baseHeight := availableHeight / rows
	extraHeight := availableHeight % rows
	if baseHeight < 4 {
		baseHeight = 4
		extraHeight = 0
	}
	tileWidth := (preview.W - gap*(cols-1)) / cols
	if tileWidth < 14 {
		tileWidth = 14
	}

	hits := make([]mouse.PaneHit, 0, len(panes))
	rowY := preview.Y
	for r := 0; r < rows; r++ {
		rowHeight := baseHeight
		if r == rows-1 {
			rowHeight += extraHeight
		}
		for c := 0; c < cols; c++ {
			idx := r*cols + c
			if idx >= len(panes) {
				continue
			}
			pane := panes[idx]
			outer := mouse.Rect{
				X: preview.X + c*tileWidth,
				Y: rowY,
				W: tileWidth,
				H: rowHeight,
			}
			borders := tileBorders{
				top:    r == 0,
				left:   c == 0,
				right:  true,
				bottom: true,
			}
			content := projectTileContentRect(outer, pane, borders)
			hits = append(hits, mouse.PaneHit{
				PaneID: pane.ID,
				Selection: mouse.Selection{
					Project: project.Name,
					Session: session.Name,
					Pane:    pane.Index,
				},
				Outer:   outer,
				Content: content,
			})
		}
		rowY += rowHeight
	}
	return hits
}

func dashboardPaneContentRect(outer mouse.Rect, previewLines int) mouse.Rect {
	inner := tileInnerRect(outer, tileBorders{top: true, right: true, bottom: true, left: true})
	if inner.Empty() {
		return mouse.Rect{}
	}
	available := previewLines
	if inner.H-2 < available {
		available = inner.H - 2
	}
	if available <= 0 {
		return mouse.Rect{}
	}
	return mouse.Rect{X: inner.X, Y: inner.Y + 2, W: inner.W, H: available}
}

func projectTileContentRect(outer mouse.Rect, pane PaneItem, borders tileBorders) mouse.Rect {
	inner := tileInnerRect(outer, borders)
	if inner.Empty() {
		return mouse.Rect{}
	}
	headerLines := 1
	if strings.TrimSpace(pane.Command) != "" {
		headerLines++
	}
	available := inner.H - headerLines
	if available <= 0 {
		return mouse.Rect{}
	}
	return mouse.Rect{X: inner.X, Y: inner.Y + headerLines, W: inner.W, H: available}
}

func tileInnerRect(outer mouse.Rect, borders tileBorders) mouse.Rect {
	padLeft, padRight := 1, 1
	padTop, padBottom := 0, 0
	left := boolToInt(borders.left)
	right := boolToInt(borders.right)
	top := boolToInt(borders.top)
	bottom := boolToInt(borders.bottom)

	inner := mouse.Rect{
		X: outer.X + left + padLeft,
		Y: outer.Y + top + padTop,
		W: outer.W - left - right - padLeft - padRight,
		H: outer.H - top - bottom - padTop - padBottom,
	}
	if inner.W < 0 {
		inner.W = 0
	}
	if inner.H < 0 {
		inner.H = 0
	}
	return inner
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
