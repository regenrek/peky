package app

import (
	"context"
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
	if _, handled := m.handleQuickReplyClick(msg); handled {
		m.updateTerminalMouseDrag(msg)
		return m, cursorCmd
	}
	m.updateTerminalMouseDrag(msg)
	if cmd, handled := m.handleOfflineScrollWheel(msg); handled {
		return m, tea.Batch(cursorCmd, cmd)
	}
	cmd := m.mouse.UpdateDashboard(msg, mouse.DashboardCallbacks{
		HitHeader:             m.hitTestHeader,
		HitPane:               m.hitTestPane,
		HitIsSelected:         m.hitIsSelected,
		ApplySelection:        m.applySelectionFromHit,
		SelectDashboardTab:    m.selectDashboardTab,
		SelectProjectTab:      m.selectProjectTab,
		OpenProjectPicker:     m.openProjectPicker,
		SetTerminalFocus:      m.setTerminalFocus,
		TerminalFocus:         func() bool { return m.terminalFocus },
		SupportsTerminalFocus: m.supportsTerminalFocus,
		SelectionCmd:          m.selectionCmd,
		SelectionRefreshCmd:   m.selectionRefreshCmd,
		RefreshPaneViewsCmd:   m.refreshPaneViewsCmd,
		ForwardMouseEvent:     m.forwardMouseEvent,
		FocusUnavailable: func() {
			m.setToast("Terminal focus is only available for PeakyPanes-managed sessions", toastInfo)
		},
	})
	return m, tea.Batch(cursorCmd, cmd)
}

func (m *Model) handleQuickReplyClick(msg tea.MouseMsg) (tea.Cmd, bool) {
	if m == nil {
		return nil, false
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil, false
	}
	rect, ok := m.quickReplyRect()
	if !ok || !rect.Contains(msg.X, msg.Y) {
		return nil, false
	}
	if m.terminalFocus {
		m.setTerminalFocus(false)
	}
	if m.filterActive {
		m.filterActive = false
		m.filterInput.Blur()
	}
	m.quickReplyInput.Focus()
	return nil, true
}

func (m *Model) allowMouseMotion() bool {
	if m == nil {
		return false
	}
	if m.state != StateDashboard {
		return false
	}
	if !m.terminalFocus {
		return true
	}
	if !m.supportsTerminalFocus() {
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

func (m *Model) applySelectionFromHit(sel mouse.Selection) bool {
	appSelection := selectionFromMouse(sel)
	if m.selection == appSelection {
		return false
	}
	m.applySelection(appSelection)
	m.selectionVersion++
	return true
}

func (m *Model) selectionCmd() tea.Cmd {
	if m.tab == TabDashboard {
		return nil
	}
	return m.selectionRefreshCmd()
}

func (m *Model) hitIsSelected(hit mouse.PaneHit) bool {
	pane := m.selectedPane()
	if pane == nil {
		return false
	}
	if strings.TrimSpace(pane.ID) != "" {
		return pane.ID == hit.PaneID
	}
	return m.selection == selectionFromMouse(hit.Selection)
}

func (m *Model) forwardMouseEvent(hit mouse.PaneHit, msg tea.MouseMsg) tea.Cmd {
	if m == nil || m.client == nil {
		return nil
	}
	if !m.supportsTerminalFocus() {
		return nil
	}
	paneID, payload, ok := m.mouseForwardPayload(hit, msg)
	if !ok {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), terminalActionTimeout)
		defer cancel()
		if err := m.client.SendMouse(ctx, paneID, payload); err != nil {
			return ErrorMsg{Err: err, Context: "send mouse"}
		}
		return nil
	}
}

func (m *Model) mouseForwardPayload(hit mouse.PaneHit, msg tea.MouseMsg) (string, sessiond.MousePayload, bool) {
	paneID, relX, relY, ok := m.mouseForwardTarget(hit, msg)
	if !ok {
		return "", sessiond.MousePayload{}, false
	}
	payload, ok := mousePayloadFromTea(msg, relX, relY)
	if !ok {
		return "", sessiond.MousePayload{}, false
	}
	payload.Route = m.mouseRouteForForward()
	if !m.allowMousePayload(paneID, payload) {
		return "", sessiond.MousePayload{}, false
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
	if m.terminalFocus {
		return sessiond.MouseRouteAuto
	}
	return sessiond.MouseRouteHostSelection
}

func (m *Model) allowMousePayload(paneID string, payload sessiond.MousePayload) bool {
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
	if m.state != StateDashboard || !m.supportsTerminalFocus() {
		m.terminalMouseDrag = false
		return
	}
	hit, ok := m.hitTestPane(msg.X, msg.Y)
	if !ok || !hit.Content.Contains(msg.X, msg.Y) {
		m.terminalMouseDrag = false
		return
	}
	if m.terminalFocus && !m.hitIsSelected(hit) {
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
