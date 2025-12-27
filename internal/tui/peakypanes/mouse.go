package peakypanes

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"
)

const doubleClickThreshold = 350 * time.Millisecond

type mouseState struct {
	lastClickAt     time.Time
	lastClickPaneID string
	lastClickButton tea.MouseButton
}

type paneHit struct {
	PaneID    string
	Selection selectionState
	Outer     rect
	Content   rect
}

func (m *Model) updateDashboardMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	hit, ok := m.hitTestPane(msg.X, msg.Y)

	if m.terminalFocus {
		if ok && m.hitIsSelected(hit) && hit.Content.contains(msg.X, msg.Y) {
			m.forwardMouseEvent(hit, msg)
			return m, nil
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && ok && !m.hitIsSelected(hit) {
			m.setTerminalFocus(false)
			changed := m.applySelectionFromHit(hit.Selection)
			m.recordClick(hit, msg)
			if changed {
				return m, m.selectionCmd()
			}
			return m, nil
		}
		return m, nil
	}

	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	if !ok {
		return m, nil
	}

	if m.isDoubleClick(hit, msg) {
		m.clearLastClick()
		changed := m.applySelectionFromHit(hit.Selection)
		if !m.supportsTerminalFocus() {
			m.setToast("Terminal focus is only available for PeakyPanes-managed sessions", toastInfo)
			if changed {
				return m, m.selectionCmd()
			}
			return m, nil
		}
		m.setTerminalFocus(true)
		if changed {
			return m, m.selectionCmd()
		}
		return m, nil
	}

	m.recordClick(hit, msg)
	if m.applySelectionFromHit(hit.Selection) {
		return m, m.selectionCmd()
	}
	return m, nil
}

func (m *Model) allowMouseMotion() bool {
	if m == nil {
		return false
	}
	if m.state != StateDashboard || !m.terminalFocus {
		return false
	}
	if !m.supportsTerminalFocus() {
		return false
	}
	win := m.nativeFocusedWindow()
	if win == nil {
		return false
	}
	return win.AllowsMouseMotion()
}

func (m *Model) applySelectionFromHit(sel selectionState) bool {
	if m.selection == sel {
		return false
	}
	m.applySelection(sel)
	m.selectionVersion++
	return true
}

func (m *Model) selectionCmd() tea.Cmd {
	return m.selectionRefreshCmd()
}

func (m *Model) recordClick(hit paneHit, msg tea.MouseMsg) {
	m.mouse.lastClickAt = time.Now()
	m.mouse.lastClickPaneID = hit.PaneID
	m.mouse.lastClickButton = msg.Button
}

func (m *Model) clearLastClick() {
	m.mouse.lastClickAt = time.Time{}
	m.mouse.lastClickPaneID = ""
	m.mouse.lastClickButton = tea.MouseButtonNone
}

func (m *Model) isDoubleClick(hit paneHit, msg tea.MouseMsg) bool {
	if hit.PaneID == "" {
		return false
	}
	if m.mouse.lastClickPaneID != hit.PaneID {
		return false
	}
	if m.mouse.lastClickButton != msg.Button {
		return false
	}
	if time.Since(m.mouse.lastClickAt) > doubleClickThreshold {
		return false
	}
	return true
}

func (m *Model) hitIsSelected(hit paneHit) bool {
	pane := m.selectedPane()
	if pane == nil {
		return false
	}
	if strings.TrimSpace(pane.ID) != "" {
		return pane.ID == hit.PaneID
	}
	return m.selection == hit.Selection
}

func (m *Model) forwardMouseEvent(hit paneHit, msg tea.MouseMsg) {
	if m.native == nil {
		return
	}
	if !m.supportsTerminalFocus() {
		return
	}
	if strings.TrimSpace(hit.PaneID) == "" {
		return
	}
	if !hit.Content.contains(msg.X, msg.Y) {
		return
	}
	relX := msg.X - hit.Content.X
	relY := msg.Y - hit.Content.Y
	if relX < 0 || relY < 0 {
		return
	}

	event, ok := mouseEventFromTea(msg, relX, relY)
	if !ok {
		return
	}
	win := m.native.Window(hit.PaneID)
	if win == nil {
		return
	}
	if _, isMotion := event.(uv.MouseMotionEvent); isMotion && !win.AllowsMouseMotion() {
		return
	}
	if err := m.native.SendMouse(hit.PaneID, event); err != nil {
		m.setToast("Mouse input failed: "+err.Error(), toastError)
	}
}

func (m *Model) hitTestPane(x, y int) (paneHit, bool) {
	for _, hit := range m.paneHits() {
		if hit.Outer.contains(x, y) {
			return hit, true
		}
	}
	return paneHit{}, false
}

func (m *Model) paneHits() []paneHit {
	if m.state != StateDashboard {
		return nil
	}
	if m.tab == TabProject {
		return m.projectPaneHits()
	}
	return m.dashboardPaneHits()
}

func (m *Model) dashboardPaneHits() []paneHit {
	body, ok := m.dashboardBodyRect()
	if !ok {
		return nil
	}
	columns := collectDashboardColumns(m.data.Projects)
	if len(columns) == 0 {
		return nil
	}
	columns = m.filteredDashboardColumns(columns)

	totalPanes := 0
	for _, column := range columns {
		totalPanes += len(column.Panes)
	}
	if totalPanes == 0 {
		return nil
	}

	selectedProject := m.dashboardSelectedProject(columns)
	previewLines := dashboardPreviewLines(m.settings)

	gap := 2
	minColWidth := 24
	maxCols := (body.W + gap) / (minColWidth + gap)
	if maxCols < 1 {
		maxCols = 1
	}

	selectedIndex := 0
	for i, column := range columns {
		if column.ProjectName == selectedProject {
			selectedIndex = i
			break
		}
	}
	if len(columns) > maxCols {
		start := selectedIndex - maxCols/2
		if start < 0 {
			start = 0
		}
		if start+maxCols > len(columns) {
			start = len(columns) - maxCols
		}
		columns = columns[start : start+maxCols]
		selectedIndex = selectedIndex - start
	}

	colWidth := body.W
	if len(columns) > 1 {
		colWidth = (body.W - gap*(len(columns)-1)) / len(columns)
	}
	if colWidth < 1 {
		colWidth = 1
	}

	hits := make([]paneHit, 0)
	for i, column := range columns {
		if len(column.Panes) == 0 {
			continue
		}
		colX := body.X + i*(colWidth+gap)

		headerHeight := 3
		bodyHeight := body.H - headerHeight
		if bodyHeight <= 0 {
			continue
		}

		blockHeight := dashboardPaneBlockHeight(previewLines)
		if blockHeight > bodyHeight {
			blockHeight = bodyHeight
		}
		if blockHeight < 3 {
			blockHeight = bodyHeight
		}
		visibleBlocks := bodyHeight / blockHeight
		if visibleBlocks < 1 {
			visibleBlocks = 1
		}

		selectedPaneIndex := -1
		if i == selectedIndex {
			selectedPaneIndex = dashboardPaneIndex(column.Panes, m.selection)
		}
		start := 0
		if i == selectedIndex && selectedPaneIndex >= 0 && selectedPaneIndex >= visibleBlocks {
			start = selectedPaneIndex - visibleBlocks + 1
		}
		if start < 0 {
			start = 0
		}
		if start > len(column.Panes)-1 {
			start = len(column.Panes) - 1
		}
		end := start + visibleBlocks
		if end > len(column.Panes) {
			end = len(column.Panes)
		}
		if end < start {
			end = start
		}

		for idx := start; idx < end; idx++ {
			pane := column.Panes[idx]
			outer := rect{
				X: colX,
				Y: body.Y + headerHeight + (idx-start)*blockHeight,
				W: colWidth,
				H: blockHeight,
			}
			content := dashboardPaneContentRect(outer, previewLines)
			hits = append(hits, paneHit{
				PaneID: pane.Pane.ID,
				Selection: selectionState{
					Project: column.ProjectName,
					Session: pane.SessionName,
					Window:  pane.WindowIndex,
					Pane:    pane.Pane.Index,
				},
				Outer:   outer,
				Content: content,
			})
		}
	}
	return hits
}

func (m *Model) projectPaneHits() []paneHit {
	body, ok := m.dashboardBodyRect()
	if !ok {
		return nil
	}
	project := m.selectedProject()
	session := m.selectedSession()
	if project == nil || session == nil {
		return nil
	}
	window := selectedWindow(session, m.selection.Window)
	if window == nil || len(window.Panes) == 0 {
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

	preview := rect{
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
		return projectPaneLayoutHits(project, session, window, preview)
	}
	return projectPaneTileHits(project, session, window, preview)
}

func projectPaneLayoutHits(project *ProjectGroup, session *SessionItem, window *WindowItem, preview rect) []paneHit {
	maxW, maxH := paneBounds(window.Panes)
	if maxW == 0 || maxH == 0 {
		return nil
	}

	hits := make([]paneHit, 0, len(window.Panes))
	for _, pane := range window.Panes {
		x1, y1, w, h := scalePane(pane, maxW, maxH, preview.W, preview.H)
		if w <= 0 || h <= 0 {
			continue
		}
		outer := rect{
			X: preview.X + x1,
			Y: preview.Y + y1,
			W: w,
			H: h,
		}
		hits = append(hits, paneHit{
			PaneID: pane.ID,
			Selection: selectionState{
				Project: project.Name,
				Session: session.Name,
				Window:  window.Index,
				Pane:    pane.Index,
			},
			Outer:   outer,
			Content: rect{},
		})
	}
	return hits
}

func projectPaneTileHits(project *ProjectGroup, session *SessionItem, window *WindowItem, preview rect) []paneHit {
	panes := window.Panes
	if len(panes) == 0 {
		return nil
	}

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

	hits := make([]paneHit, 0, len(panes))
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
			outer := rect{
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
			hits = append(hits, paneHit{
				PaneID: pane.ID,
				Selection: selectionState{
					Project: project.Name,
					Session: session.Name,
					Window:  window.Index,
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

func dashboardPaneContentRect(outer rect, previewLines int) rect {
	inner := tileInnerRect(outer, tileBorders{top: true, right: true, bottom: true, left: true})
	if inner.empty() {
		return rect{}
	}
	available := previewLines
	if inner.H-2 < available {
		available = inner.H - 2
	}
	if available <= 0 {
		return rect{}
	}
	return rect{X: inner.X, Y: inner.Y + 2, W: inner.W, H: available}
}

func projectTileContentRect(outer rect, pane PaneItem, borders tileBorders) rect {
	inner := tileInnerRect(outer, borders)
	if inner.empty() {
		return rect{}
	}
	headerLines := 1
	if strings.TrimSpace(pane.Command) != "" {
		headerLines++
	}
	available := inner.H - headerLines
	if available <= 0 {
		return rect{}
	}
	return rect{X: inner.X, Y: inner.Y + headerLines, W: inner.W, H: available}
}

func tileInnerRect(outer rect, borders tileBorders) rect {
	padLeft, padRight := 1, 1
	padTop, padBottom := 0, 0
	left := boolToInt(borders.left)
	right := boolToInt(borders.right)
	top := boolToInt(borders.top)
	bottom := boolToInt(borders.bottom)

	inner := rect{
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

func mouseEventFromTea(msg tea.MouseMsg, x, y int) (uv.MouseEvent, bool) {
	if x < 0 || y < 0 {
		return nil, false
	}
	mod := uv.KeyMod(0)
	if msg.Shift {
		mod |= uv.ModShift
	}
	if msg.Alt {
		mod |= uv.ModAlt
	}
	if msg.Ctrl {
		mod |= uv.ModCtrl
	}
	mouse := uv.Mouse{X: x, Y: y, Button: uv.MouseButton(msg.Button), Mod: mod}

	if isWheelButton(msg.Button) {
		return uv.MouseWheelEvent(mouse), true
	}
	switch msg.Action {
	case tea.MouseActionPress:
		return uv.MouseClickEvent(mouse), true
	case tea.MouseActionRelease:
		return uv.MouseReleaseEvent(mouse), true
	case tea.MouseActionMotion:
		return uv.MouseMotionEvent(mouse), true
	default:
		return nil, false
	}
}

func isWheelButton(button tea.MouseButton) bool {
	switch button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown, tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		return true
	default:
		return false
	}
}
