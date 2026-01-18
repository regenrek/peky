package app

import (
	tea "github.com/charmbracelet/bubbletea"

	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
)

func (m *Model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.updateDashboardInput(keyMsgFromTea(msg))
}

func (m *Model) updateDashboardInput(msg tuiinput.KeyMsg) (tea.Model, tea.Cmd) {
	teaMsg := msg.Tea()
	m.ensureQuickReplyBlur()
	if cmd, handled := m.handleDashboardPreInput(msg, teaMsg); handled {
		return m, cmd
	}
	if m.hardRaw {
		return m, m.sendDashboardKeyToPane(msg)
	}
	if cmd, handled := m.handleDashboardPostInput(msg, teaMsg); handled {
		return m, cmd
	}
	if m.quickReplyInput.Focused() {
		return m.updateQuickReplyInput(msg)
	}
	return m, m.sendDashboardKeyToPane(msg)
}

func (m *Model) ensureQuickReplyBlur() {
	if m == nil {
		return
	}
	if !m.quickReplyEnabled() && m.quickReplyInput.Focused() {
		m.quickReplyMouseSel.clear()
		m.resetQuickReplyHistory()
		m.resetQuickReplyMenu()
		m.quickReplyInput.Blur()
	}
}

func (m *Model) handleDashboardPreInput(msg tuiinput.KeyMsg, teaMsg tea.KeyMsg) (tea.Cmd, bool) {
	if cmd, handled := m.handleHardRawToggle(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleDashboardFilter(teaMsg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleOfflineScrollInput(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleFocusAction(msg); handled {
		return cmd, true
	}
	return nil, false
}

func (m *Model) handleDashboardPostInput(msg tuiinput.KeyMsg, teaMsg tea.KeyMsg) (tea.Cmd, bool) {
	if cmd, handled := m.handleContextMenuKey(teaMsg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleResizeKey(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleDashboardNavigation(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleSidebarToggle(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleDashboardActions(msg); handled {
		return cmd, true
	}
	return nil, false
}

func (m *Model) handleDashboardNavigation(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	if cmd, handled := m.handleProjectNav(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.handleSessionNav(msg); handled {
		return cmd, true
	}
	if cmd, handled := m.handlePaneNav(msg); handled {
		return cmd, true
	}
	m.applyTogglePanes(msg)
	return nil, false
}

func (m *Model) handleFocusAction(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	if m == nil || m.keys == nil {
		return nil, false
	}
	if !matchesBinding(msg, m.keys.focusAction) {
		return nil, false
	}
	if !m.quickReplyEnabled() {
		return nil, true
	}
	if m.quickReplyInput.Focused() {
		m.quickReplyMouseSel.clear()
		m.resetQuickReplyHistory()
		m.resetQuickReplyMenu()
		m.quickReplyInput.Blur()
		return nil, true
	}
	return m.prepareQuickReplyInput(), true
}

func (m *Model) handleDashboardFilter(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.filterActive {
		return nil, false
	}
	switch msg.String() {
	case "enter":
		m.filterActive = false
		m.filterInput.Blur()
		return nil, true
	case "esc":
		m.filterActive = false
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		return nil, true
	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return cmd, true
	}
}

func (m *Model) handleHardRawToggle(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	if m == nil || m.keys == nil {
		return nil, false
	}
	if !matchesBinding(msg, m.keys.hardRaw) {
		return nil, false
	}
	return m.toggleHardRaw(), true
}

func (m *Model) sendDashboardKeyToPane(msg tuiinput.KeyMsg) tea.Cmd {
	if m == nil {
		return nil
	}
	if m.keys != nil {
		teaMsg := msg.Tea()
		if matchesBinding(msg, m.keys.scrollback) || matchesBinding(msg, m.keys.copyMode) {
			if cmd := m.handleTerminalKeyCmd(teaMsg); cmd != nil {
				return cmd
			}
		}
	}
	payload := encodeKeyMsg(msg.Tea())
	if len(payload) == 0 {
		return nil
	}
	return m.sendPaneInputCmd(payload, "send to pane")
}

func (m *Model) handleProjectNav(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	switch {
	case matchesBinding(msg, m.keys.projectLeft):
		m.selectTab(-1)
		return m.selectionRefreshCmd(), true
	case matchesBinding(msg, m.keys.projectRight):
		m.selectTab(1)
		return m.selectionRefreshCmd(), true
	default:
		return nil, false
	}
}

func (m *Model) handleSessionNav(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	switch {
	case matchesBinding(msg, m.keys.sessionUp):
		if m.tab == TabDashboard {
			m.selectDashboardPane(-1)
			return nil, true
		}
		m.selectSessionOrPane(-1)
		return m.selectionRefreshCmd(), true
	case matchesBinding(msg, m.keys.sessionDown):
		if m.tab == TabDashboard {
			m.selectDashboardPane(1)
			return nil, true
		}
		m.selectSessionOrPane(1)
		return m.selectionRefreshCmd(), true
	case matchesBinding(msg, m.keys.sessionOnlyUp):
		if m.tab == TabDashboard {
			m.selectDashboardPane(-1)
			return nil, true
		}
		m.selectSession(-1)
		return m.selectionRefreshCmd(), true
	case matchesBinding(msg, m.keys.sessionOnlyDown):
		if m.tab == TabDashboard {
			m.selectDashboardPane(1)
			return nil, true
		}
		m.selectSession(1)
		return m.selectionRefreshCmd(), true
	default:
		return nil, false
	}
}

func (m *Model) handlePaneNav(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	switch {
	case matchesBinding(msg, m.keys.paneNext):
		if m.tab == TabDashboard {
			m.selectDashboardProject(1)
			return nil, true
		}
		return m.cyclePane(1), true
	case matchesBinding(msg, m.keys.panePrev):
		if m.tab == TabDashboard {
			m.selectDashboardProject(-1)
			return nil, true
		}
		return m.cyclePane(-1), true
	case matchesBinding(msg, m.keys.toggleLastPane):
		m.toggleLastPane()
		return nil, true
	default:
		return nil, false
	}
}

func (m *Model) applyTogglePanes(msg tuiinput.KeyMsg) {
	if matchesBinding(msg, m.keys.togglePanes) {
		m.togglePanes()
	}
}

func (m *Model) handleSidebarToggle(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	if !matchesBinding(msg, m.keys.toggleSidebar) {
		return nil, false
	}
	m.toggleSidebar()
	return nil, true
}

func (m *Model) handleDashboardActions(msg tuiinput.KeyMsg) (tea.Cmd, bool) {
	switch {
	case matchesBinding(msg, m.keys.newSession):
		m.openLayoutPicker()
		return nil, true
	case matchesBinding(msg, m.keys.openProject):
		m.openProjectPicker()
		return nil, true
	case matchesBinding(msg, m.keys.commandPalette):
		return m.openCommandPalette(), true
	case matchesBinding(msg, m.keys.refresh):
		m.setToast("Refreshing...", toastInfo)
		return m.requestRefreshCmd(), true
	case matchesBinding(msg, m.keys.editConfig):
		return m.editConfig(), true
	case matchesBinding(msg, m.keys.kill):
		m.openKillConfirm()
		return nil, true
	case matchesBinding(msg, m.keys.closeProject):
		m.openCloseProjectConfirm()
		return nil, true
	case matchesBinding(msg, m.keys.filter):
		m.filterActive = true
		m.filterInput.Focus()
		m.quickReplyInput.Blur()
		return nil, true
	case matchesBinding(msg, m.keys.help):
		m.setState(StateHelp)
		return nil, true
	case matchesBinding(msg, m.keys.quit):
		return m.requestQuit(), true
	default:
		return nil, false
	}
}
