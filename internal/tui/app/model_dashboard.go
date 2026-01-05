package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if cmd, handled := m.handleDashboardFilter(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleOfflineScrollInput(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleTerminalFocusInput(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleTerminalFocusToggle(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleProjectNav(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleSessionNav(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handlePaneNav(msg); handled {
		return m, cmd
	}
	m.applyTogglePanes(msg)
	if cmd, handled := m.handleSidebarToggle(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleDashboardActions(msg); handled {
		return m, cmd
	}

	return m.updateQuickReply(msg)
}

func (m *Model) handleDashboardFilter(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.filterActive {
		return nil, false
	}
	switch msg.String() {
	case "enter":
		m.filterActive = false
		m.filterInput.Blur()
		m.quickReplyInput.Focus()
		return nil, true
	case "esc":
		m.filterActive = false
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		m.quickReplyInput.Focus()
		return nil, true
	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return cmd, true
	}
}

func (m *Model) handleTerminalFocusInput(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.supportsTerminalFocus() || !m.terminalFocus {
		return nil, false
	}
	if key.Matches(msg, m.keys.terminalFocus) {
		m.setTerminalFocus(false)
		return m.refreshPaneViewsCmd(), true
	}
	if cmd := m.handleTerminalKeyCmd(msg); cmd != nil {
		return cmd, true
	}
	payload := encodeKeyMsg(msg)
	if len(payload) == 0 {
		return nil, true
	}
	return m.sendPaneInputCmd(payload, "send to pane"), true
}

func (m *Model) handleTerminalFocusToggle(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !key.Matches(msg, m.keys.terminalFocus) {
		return nil, false
	}
	if !m.supportsTerminalFocus() {
		m.setToast("Terminal focus is only available for PeakyPanes-managed sessions", toastInfo)
		return nil, true
	}
	m.setTerminalFocus(!m.terminalFocus)
	return m.refreshPaneViewsCmd(), true
}

func (m *Model) handleProjectNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.projectLeft):
		m.selectTab(-1)
		return m.selectionRefreshCmd(), true
	case key.Matches(msg, m.keys.projectRight):
		m.selectTab(1)
		return m.selectionRefreshCmd(), true
	default:
		return nil, false
	}
}

func (m *Model) handleSessionNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.sessionUp):
		if m.tab == TabDashboard {
			m.selectDashboardPane(-1)
			return nil, true
		}
		m.selectSessionOrPane(-1)
		return m.selectionRefreshCmd(), true
	case key.Matches(msg, m.keys.sessionDown):
		if m.tab == TabDashboard {
			m.selectDashboardPane(1)
			return nil, true
		}
		m.selectSessionOrPane(1)
		return m.selectionRefreshCmd(), true
	case key.Matches(msg, m.keys.sessionOnlyUp):
		if m.tab == TabDashboard {
			m.selectDashboardPane(-1)
			return nil, true
		}
		m.selectSession(-1)
		return m.selectionRefreshCmd(), true
	case key.Matches(msg, m.keys.sessionOnlyDown):
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

func (m *Model) handlePaneNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.paneNext):
		if m.tab == TabDashboard {
			m.selectDashboardProject(1)
			return nil, true
		}
		return m.cyclePane(1), true
	case key.Matches(msg, m.keys.panePrev):
		if m.tab == TabDashboard {
			m.selectDashboardProject(-1)
			return nil, true
		}
		return m.cyclePane(-1), true
	default:
		return nil, false
	}
}

func (m *Model) applyTogglePanes(msg tea.KeyMsg) {
	if key.Matches(msg, m.keys.togglePanes) {
		m.togglePanes()
	}
}

func (m *Model) handleSidebarToggle(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !key.Matches(msg, m.keys.toggleSidebar) {
		return nil, false
	}
	m.toggleSidebar()
	return nil, true
}

func (m *Model) handleDashboardActions(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.newSession):
		m.openLayoutPicker()
		return nil, true
	case key.Matches(msg, m.keys.openProject):
		m.openProjectPicker()
		return nil, true
	case key.Matches(msg, m.keys.commandPalette):
		return m.openCommandPalette(), true
	case key.Matches(msg, m.keys.refresh):
		m.setToast("Refreshing...", toastInfo)
		return m.requestRefreshCmd(), true
	case key.Matches(msg, m.keys.editConfig):
		return m.editConfig(), true
	case key.Matches(msg, m.keys.kill):
		m.openKillConfirm()
		return nil, true
	case key.Matches(msg, m.keys.closeProject):
		m.openCloseProjectConfirm()
		return nil, true
	case key.Matches(msg, m.keys.filter):
		m.filterActive = true
		m.filterInput.Focus()
		m.quickReplyInput.Blur()
		return nil, true
	case key.Matches(msg, m.keys.help):
		m.setState(StateHelp)
		return nil, true
	case key.Matches(msg, m.keys.quit):
		return m.requestQuit(), true
	default:
		return nil, false
	}
}
