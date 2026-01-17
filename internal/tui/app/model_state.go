package app

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) setState(state ViewState) {
	m.state = state
	if state == StateDashboard {
		return
	}
	m.filterActive = false
	m.filterInput.Blur()
	m.quickReplyInput.Blur()
}

func (m *Model) toggleHardRaw() tea.Cmd {
	if m == nil {
		return nil
	}
	return m.setHardRaw(!m.hardRaw)
}

func (m *Model) setHardRaw(enabled bool) tea.Cmd {
	if m == nil {
		return nil
	}
	if m.hardRaw == enabled {
		return nil
	}
	m.hardRaw = enabled
	var cmd tea.Cmd
	if enabled {
		if m.contextMenu.open {
			m.closeContextMenu()
		}
		if m.resize.drag.active {
			cmd = m.cancelResizeDrag()
		}
		if m.resize.mode {
			m.exitResizeMode()
		}
		if m.filterActive {
			m.filterActive = false
			m.filterInput.Blur()
		}
		m.quickReplyInput.Blur()
	}
	refresh := m.refreshPaneViewsCmd()
	if cmd == nil {
		return refresh
	}
	return tea.Batch(cmd, refresh)
}
