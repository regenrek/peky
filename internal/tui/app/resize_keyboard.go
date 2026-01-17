package app

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
)

const (
	resizeNudgeStep     = 10
	resizeNudgeStepFast = 25
	resizeNudgeStepMax  = 50
)

func (m *Model) handleResizeKey(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	if m == nil || m.state != StateDashboard {
		return nil, false
	}
	teaMsg := msg.Tea()
	if m.resize.drag.active && teaMsg.String() == "esc" {
		return m.cancelResizeDrag(), true
	}
	if m.resize.mode {
		return m.handleResizeModeKey(teaMsg)
	}
	if m.keys != nil && matchesBinding(msg, m.keys.resizeMode) {
		if m.hardRaw {
			m.setToast("Keyboard resize needs RAW off (mouse drag works)", toastInfo)
			return nil, true
		}
		m.enterResizeMode()
		return nil, true
	}
	return nil, false
}

func (m *Model) enterResizeMode() {
	if m == nil {
		return
	}
	m.resize.mode = true
	if !m.resize.snap {
		m.resize.snap = true
	}
	m.resize.key.snapState = sessiond.SnapState{}
	if edge, ok := m.defaultResizeEdge(); ok {
		m.resize.key.edge = edge
		m.resize.key.hasEdge = true
	}
}

func (m *Model) exitResizeMode() {
	if m == nil {
		return
	}
	m.resize.mode = false
	m.resize.key = resizeKeyState{}
}

func (m *Model) handleResizeModeKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		m.exitResizeMode()
		return nil, true
	case "tab":
		m.cycleResizeEdge(1)
		return nil, true
	case "shift+tab":
		m.cycleResizeEdge(-1)
		return nil, true
	case "s":
		m.resize.snap = !m.resize.snap
		if !m.resize.snap {
			m.resize.key.snapState = sessiond.SnapState{}
		}
		return nil, true
	case "0":
		return m.resetPaneSizes(), true
	case "z":
		return m.toggleZoomPane(), true
	}

	keyStr := msg.String()
	step, axis, ok := resizeNudgeForKey(keyStr)
	if !ok {
		return nil, true
	}
	edge, ok := m.activeResizeEdge()
	if !ok {
		return nil, true
	}
	if !resizeEdgeMatchesAxis(edge.Edge, axis) {
		return nil, true
	}
	delta, ok := resizeDeltaForAxis(axis, keyStr, step)
	if !ok {
		return nil, true
	}
	return m.applyKeyboardResize(edge, delta), true
}

func resizeNudgeForKey(key string) (int, layout.Axis, bool) {
	switch key {
	case "left", "right":
		return resizeNudgeStep, layout.AxisHorizontal, true
	case "up", "down":
		return resizeNudgeStep, layout.AxisVertical, true
	case "shift+left", "shift+right":
		return resizeNudgeStepFast, layout.AxisHorizontal, true
	case "shift+up", "shift+down":
		return resizeNudgeStepFast, layout.AxisVertical, true
	case "ctrl+left", "ctrl+right":
		return resizeNudgeStepMax, layout.AxisHorizontal, true
	case "ctrl+up", "ctrl+down":
		return resizeNudgeStepMax, layout.AxisVertical, true
	default:
		return 0, layout.AxisHorizontal, false
	}
}

func resizeEdgeMatchesAxis(edge sessiond.ResizeEdge, axis layout.Axis) bool {
	switch axis {
	case layout.AxisHorizontal:
		return edge == sessiond.ResizeEdgeLeft || edge == sessiond.ResizeEdgeRight
	case layout.AxisVertical:
		return edge == sessiond.ResizeEdgeUp || edge == sessiond.ResizeEdgeDown
	default:
		return false
	}
}

func resizeDeltaForAxis(axis layout.Axis, key string, step int) (int, bool) {
	if step == 0 {
		return 0, false
	}
	switch axis {
	case layout.AxisHorizontal:
		if key == "left" || strings.Contains(key, "left") {
			return -step, true
		}
		if key == "right" || strings.Contains(key, "right") {
			return step, true
		}
	case layout.AxisVertical:
		if key == "up" || strings.Contains(key, "up") {
			return -step, true
		}
		if key == "down" || strings.Contains(key, "down") {
			return step, true
		}
	}
	return 0, false
}

func (m *Model) defaultResizeEdge() (resizeEdgeRef, bool) {
	if m.resize.hover.hasEdge {
		return m.resize.hover.edge, true
	}
	if m.resize.hover.hasCorner {
		return m.resize.hover.corner.Vertical, true
	}
	return m.firstEdgeForSelection()
}

func (m *Model) activeResizeEdge() (resizeEdgeRef, bool) {
	if m.resize.key.hasEdge {
		return m.resize.key.edge, true
	}
	return m.firstEdgeForSelection()
}

func (m *Model) firstEdgeForSelection() (resizeEdgeRef, bool) {
	pane := m.selectedPane()
	if pane == nil || pane.ID == "" {
		return resizeEdgeRef{}, false
	}
	edges := m.resizeEdgesForPane(pane.ID)
	if len(edges) == 0 {
		return resizeEdgeRef{}, false
	}
	return edges[0], true
}

func (m *Model) cycleResizeEdge(dir int) {
	pane := m.selectedPane()
	if pane == nil || pane.ID == "" {
		return
	}
	edges := m.resizeEdgesForPane(pane.ID)
	if len(edges) == 0 {
		return
	}
	current := -1
	if m.resize.key.hasEdge {
		for i, edge := range edges {
			if edge == m.resize.key.edge {
				current = i
				break
			}
		}
	}
	if current == -1 {
		m.resize.key.edge = edges[0]
		m.resize.key.hasEdge = true
		return
	}
	next := current + dir
	for next < 0 {
		next += len(edges)
	}
	next = next % len(edges)
	m.resize.key.edge = edges[next]
	m.resize.key.hasEdge = true
}

func (m *Model) resizeEdgesForPane(paneID string) []resizeEdgeRef {
	geom, ok := m.resizeGeometry()
	if !ok {
		return nil
	}
	var paneRect layout.Rect
	found := false
	for _, pane := range geom.Panes {
		if pane.ID == paneID {
			paneRect = pane.Layout
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	edges := make([]resizeEdgeRef, 0, 4)
	for _, edge := range geom.Edges {
		switch {
		case edge.LayoutPos == paneRect.X && rangesOverlap(edge.RangeStart, edge.RangeEnd, paneRect.Y, paneRect.Y+paneRect.H):
			edges = append(edges, resizeEdgeRef{PaneID: paneID, Edge: sessiond.ResizeEdgeLeft})
		case edge.LayoutPos == paneRect.X+paneRect.W && rangesOverlap(edge.RangeStart, edge.RangeEnd, paneRect.Y, paneRect.Y+paneRect.H):
			edges = append(edges, resizeEdgeRef{PaneID: paneID, Edge: sessiond.ResizeEdgeRight})
		case edge.LayoutPos == paneRect.Y && rangesOverlap(edge.RangeStart, edge.RangeEnd, paneRect.X, paneRect.X+paneRect.W):
			edges = append(edges, resizeEdgeRef{PaneID: paneID, Edge: sessiond.ResizeEdgeUp})
		case edge.LayoutPos == paneRect.Y+paneRect.H && rangesOverlap(edge.RangeStart, edge.RangeEnd, paneRect.X, paneRect.X+paneRect.W):
			edges = append(edges, resizeEdgeRef{PaneID: paneID, Edge: sessiond.ResizeEdgeDown})
		}
	}
	return uniqueEdges(edges)
}

func uniqueEdges(edges []resizeEdgeRef) []resizeEdgeRef {
	if len(edges) < 2 {
		return edges
	}
	seen := make(map[resizeEdgeRef]struct{}, len(edges))
	out := make([]resizeEdgeRef, 0, len(edges))
	for _, edge := range edges {
		if _, ok := seen[edge]; ok {
			continue
		}
		seen[edge] = struct{}{}
		out = append(out, edge)
	}
	return out
}

func rangesOverlap(startA, endA, startB, endB int) bool {
	if endA < startA {
		startA, endA = endA, startA
	}
	if endB < startB {
		startB, endB = endB, startB
	}
	return startA < endB && startB < endA
}

func (m *Model) applyKeyboardResize(edge resizeEdgeRef, delta int) tea.Cmd {
	session := m.selectedSession()
	if session == nil || session.Name == "" || m.client == nil {
		return nil
	}
	engine := m.layoutEngines[session.Name]
	if engine == nil {
		return nil
	}
	result, err := applyResizeToEngine(engine, edge, delta, m.resize.snap, m.resize.key.snapState)
	if err != nil {
		m.setToast("Resize failed: "+err.Error(), toastError)
		return nil
	}
	m.resize.key.snapState = result.snapState
	m.layoutEngineVersion++
	m.resize.invalidateCache()
	client := m.client
	sessionName := session.Name
	paneID := edge.PaneID
	edgeDir := edge.Edge
	snapEnabled := m.resize.snap
	snapState := m.resize.key.snapState
	return func() tea.Msg {
		if client == nil {
			return ErrorMsg{Err: errors.New("session client unavailable"), Context: "resize pane"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if _, err := client.ResizePaneEdge(ctx, sessionName, paneID, edgeDir, delta, snapEnabled, snapState); err != nil {
			return ErrorMsg{Err: err, Context: "resize pane"}
		}
		return nil
	}
}

func (m *Model) resetPaneSizes() tea.Cmd {
	session := m.selectedSession()
	pane := m.selectedPane()
	if session == nil || pane == nil {
		return nil
	}
	return m.resetPaneSizesFor(session.Name, pane.ID)
}

func (m *Model) toggleZoomPane() tea.Cmd {
	session := m.selectedSession()
	pane := m.selectedPane()
	if session == nil || pane == nil {
		return nil
	}
	return m.toggleZoomPaneFor(session.Name, pane.ID)
}

func (m *Model) resetPaneSizesFor(sessionName, paneID string) tea.Cmd {
	sessionName = strings.TrimSpace(sessionName)
	paneID = strings.TrimSpace(paneID)
	if sessionName == "" || paneID == "" || m.client == nil {
		return nil
	}
	if engine := m.layoutEngines[sessionName]; engine != nil {
		if _, err := engine.Apply(layout.ResetSizesOp{PaneID: paneID}); err == nil {
			m.layoutEngineVersion++
			m.resize.invalidateCache()
		}
	}
	client := m.client
	return func() tea.Msg {
		if client == nil {
			return ErrorMsg{Err: errors.New("session client unavailable"), Context: "reset pane sizes"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if _, err := client.ResetPaneSizes(ctx, sessionName, paneID); err != nil {
			return ErrorMsg{Err: err, Context: "reset pane sizes"}
		}
		return nil
	}
}

func (m *Model) toggleZoomPaneFor(sessionName, paneID string) tea.Cmd {
	sessionName = strings.TrimSpace(sessionName)
	paneID = strings.TrimSpace(paneID)
	if sessionName == "" || paneID == "" || m.client == nil {
		return nil
	}
	if engine := m.layoutEngines[sessionName]; engine != nil {
		if _, err := engine.Apply(layout.ZoomOp{PaneID: paneID, Toggle: true}); err == nil {
			m.layoutEngineVersion++
			m.resize.invalidateCache()
		}
	}
	client := m.client
	return func() tea.Msg {
		if client == nil {
			return ErrorMsg{Err: errors.New("session client unavailable"), Context: "zoom pane"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if _, err := client.ZoomPane(ctx, sessionName, paneID, true); err != nil {
			return ErrorMsg{Err: err, Context: "zoom pane"}
		}
		return nil
	}
}
