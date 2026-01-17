package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func selectionFromMouse(sel mouse.Selection) selectionState {
	return selectionState{
		ProjectID: sel.ProjectID,
		Session:   sel.Session,
		Pane:      sel.Pane,
	}
}

func (m *Model) updateDashboardMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	cursorCmd := m.updateCursorShape(msg)
	if cmd, handled := m.handleResizeMouse(msg); handled {
		return m, tea.Batch(cursorCmd, cmd)
	}
	if cmd, handled := m.handleContextMenuMouse(msg); handled {
		return m, tea.Batch(cursorCmd, cmd)
	}
	if cmd, handled := m.handleServerStatusClick(msg); handled {
		return m, tea.Batch(cursorCmd, cmd)
	}
	if cmd, handled := m.handleQuickReplyMouse(msg); handled {
		m.updateTerminalMouseDrag(msg)
		return m, tea.Batch(cursorCmd, cmd)
	}
	if cmd, handled := m.handlePaneTopbarClick(msg); handled {
		return m, tea.Batch(cursorCmd, cmd)
	}
	m.updateTerminalMouseDrag(msg)
	if cmd, handled := m.handleOfflineScrollWheel(msg); handled {
		return m, tea.Batch(cursorCmd, cmd)
	}
	cmd := m.mouse.UpdateDashboard(msg, mouse.DashboardCallbacks{
		HitHeader:           m.hitTestHeader,
		HitPane:             m.hitTestPane,
		ApplySelection:      m.applySelectionFromHit,
		SelectDashboardTab:  m.selectDashboardTab,
		SelectProjectTab:    m.selectProjectTab,
		OpenProjectPicker:   m.openProjectPicker,
		SelectionCmd:        m.selectionCmd,
		SelectionRefreshCmd: m.selectionRefreshCmd,
		RefreshPaneViewsCmd: m.refreshPaneViewsCmd,
		ForwardMouseEvent:   m.forwardMouseEvent,
	})
	return m, tea.Batch(cursorCmd, cmd)
}

func (m *Model) handleServerStatusClick(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil {
		return nil, false
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil, false
	}
	rect, ok := m.serverStatusRect()
	if !ok || !rect.Contains(msg.X, msg.Y) {
		return nil, false
	}
	switch m.serverStatus() {
	case "restored":
		m.setState(StateRestartNotice)
	case "down":
		m.setState(StateConfirmRestart)
	default:
		return nil, false
	}
	return nil, true
}

func (m *Model) allowMouseMotion() bool {
	if m == nil {
		return false
	}
	if m.state != StateDashboard {
		return false
	}
	if !m.hardRaw {
		return true
	}
	if m.client == nil {
		return false
	}
	pane := m.selectedPane()
	if pane == nil || strings.TrimSpace(pane.ID) == "" {
		return m.terminalMouseDrag
	}
	if m.paneMouseMotion[pane.ID] {
		return true
	}
	return m.terminalMouseDrag
}

func (m *Model) allowMouseMotionFor(msg tea.MouseMsg) bool {
	if m == nil {
		return false
	}
	if m.resize.drag.active {
		return true
	}
	if m.state == StateDashboard && m.tab == TabProject {
		if hit, ok := m.resizeHitTest(msg.X, msg.Y); ok {
			if hit.Kind == resizeHitEdge || hit.Kind == resizeHitCorner {
				return true
			}
		}
	}
	return m.allowMouseMotion()
}

func (m *Model) applySelectionFromHit(sel mouse.Selection) bool {
	appSelection := selectionFromMouse(sel)
	if m.selection == appSelection {
		return false
	}
	m.applySelection(appSelection)
	m.quickReplyMouseSel.clear()
	m.resetQuickReplyHistory()
	m.resetQuickReplyMenu()
	m.quickReplyInput.Blur()
	m.selectionVersion++
	return true
}

func (m *Model) selectionCmd() tea.Cmd {
	if m.tab == TabDashboard {
		return nil
	}
	return m.selectionRefreshCmd()
}

func (m *Model) forwardMouseEvent(hit mouse.PaneHit, msg tea.MouseMsg) tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	paneID, payload, ok := m.mouseForwardPayload(hit, msg)
	if !ok {
		return nil
	}
	if payload.Wheel {
		return m.forwardWheelEvent(paneID, payload, msg)
	}
	return m.enqueueMouseSend(paneID, payload)
}

func (m *Model) forwardWheelEvent(paneID string, payload sessiond.MouseEventPayload, msg tea.MouseMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	action := terminalActionForWheel(msg.Button)
	if action == sessiond.TerminalActionUnknown {
		return nil
	}
	if shouldSendWheelAsMouse(m, paneID, payload.Route) {
		return m.enqueueMouseSend(paneID, payload)
	}
	_, rows := m.paneSizeForFallback(paneID)
	step := terminalScrollWheelStep(rows, msg.Shift, msg.Ctrl)
	return m.enqueueTerminalScroll(paneID, action, step)
}

func terminalActionForWheel(button tea.MouseButton) sessiond.TerminalAction {
	switch button {
	case tea.MouseButtonWheelUp:
		return sessiond.TerminalScrollUp
	case tea.MouseButtonWheelDown:
		return sessiond.TerminalScrollDown
	default:
		return sessiond.TerminalActionUnknown
	}
}

func shouldSendWheelAsMouse(m *Model, paneID string, route sessiond.MouseRoute) bool {
	switch route {
	case sessiond.MouseRouteHostSelection:
		return false
	case sessiond.MouseRouteApp:
		return true
	default:
		if m == nil || m.paneHasMouse == nil {
			return false
		}
		return m.paneHasMouse[paneID]
	}
}

func terminalScrollWheelStep(rows int, shift, ctrl bool) int {
	if ctrl {
		return maxInt(1, rows-1)
	}
	if shift {
		return 1
	}
	return 3
}

func (m *Model) mouseForwardPayload(hit mouse.PaneHit, msg tea.MouseMsg) (string, sessiond.MouseEventPayload, bool) {
	paneID, relX, relY, ok := m.mouseForwardTarget(hit, msg)
	if !ok {
		return "", sessiond.MouseEventPayload{}, false
	}
	payload, ok := mousePayloadFromTea(msg, relX, relY)
	if !ok {
		return "", sessiond.MouseEventPayload{}, false
	}
	payload.Route = m.mouseRouteForForward()
	if !m.allowMousePayload(paneID, payload) {
		return "", sessiond.MouseEventPayload{}, false
	}
	return paneID, payload, true
}

func (m *Model) mouseForwardTarget(hit mouse.PaneHit, msg tea.MouseMsg) (string, int, int, bool) {
	paneID := strings.TrimSpace(hit.PaneID)
	if paneID == "" {
		return "", 0, 0, false
	}
	pane := m.paneByID(paneID)
	if pane == nil || pane.Dead || pane.Disconnected {
		return "", 0, 0, false
	}
	if !hit.Content.Contains(msg.X, msg.Y) {
		return "", 0, 0, false
	}
	relX := msg.X - hit.Content.X
	relY := msg.Y - hit.Content.Y
	if relX < 0 || relY < 0 {
		return "", 0, 0, false
	}
	return paneID, relX, relY, true
}

func (m *Model) mouseRouteForForward() sessiond.MouseRoute {
	if m.hardRaw {
		return sessiond.MouseRouteAuto
	}
	return sessiond.MouseRouteHostSelection
}

func (m *Model) allowMousePayload(paneID string, payload sessiond.MouseEventPayload) bool {
	if payload.Action != sessiond.MouseActionMotion {
		return true
	}
	if m.paneMouseMotion[paneID] {
		return true
	}
	return m.terminalMouseDrag
}

func (m *Model) updateTerminalMouseDrag(msg tea.MouseMsg) {
	if m == nil {
		return
	}
	if msg.Action == tea.MouseActionRelease {
		m.terminalMouseDrag = false
		return
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return
	}
	if m.state != StateDashboard || m.client == nil {
		m.terminalMouseDrag = false
		return
	}
	hit, ok := m.hitTestPane(msg.X, msg.Y)
	if !ok || !hit.Content.Contains(msg.X, msg.Y) {
		m.terminalMouseDrag = false
		return
	}
	m.terminalMouseDrag = true
}

func mousePayloadFromTea(msg tea.MouseMsg, x, y int) (sessiond.MouseEventPayload, bool) {
	if x < 0 || y < 0 {
		return sessiond.MouseEventPayload{}, false
	}
	var action sessiond.MouseAction
	switch msg.Action {
	case tea.MouseActionPress:
		action = sessiond.MouseActionPress
	case tea.MouseActionRelease:
		action = sessiond.MouseActionRelease
	case tea.MouseActionMotion:
		action = sessiond.MouseActionMotion
	default:
		return sessiond.MouseEventPayload{}, false
	}
	payload := sessiond.MouseEventPayload{
		X:      x,
		Y:      y,
		Button: int(msg.Button),
		Action: action,
		Shift:  msg.Shift,
		Alt:    msg.Alt,
		Ctrl:   msg.Ctrl,
		Wheel:  isWheelButton(msg.Button),
		// WheelCount is injected by the send queue when coalescing bursts.
		WheelCount: 1,
	}
	if !payload.Wheel {
		payload.WheelCount = 0
	}
	return payload, true
}

func isWheelButton(button tea.MouseButton) bool {
	switch button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown, tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		return true
	default:
		return false
	}
}
