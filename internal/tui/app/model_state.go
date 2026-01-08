package app

import "strings"

func (m *Model) setState(state ViewState) {
	m.state = state
	if state == StateDashboard && !m.filterActive && !m.terminalFocus {
		m.quickReplyInput.Focus()
	} else {
		m.quickReplyInput.Blur()
	}
}

func (m *Model) setTerminalFocus(enabled bool) {
	if m.terminalFocus == enabled {
		return
	}
	m.terminalFocus = enabled
	if enabled {
		if m.contextMenu.open {
			m.closeContextMenu()
		}
		if m.resize.mode {
			m.exitResizeMode()
		}
		if m.filterActive {
			m.filterActive = false
			m.filterInput.Blur()
		}
		m.quickReplyInput.Blur()
		return
	}
	if m.state == StateDashboard && !m.filterActive {
		m.quickReplyInput.Focus()
	}
}

func (m *Model) supportsTerminalFocus() bool {
	if m.client == nil {
		return false
	}
	pane := m.selectedPane()
	if pane == nil {
		return false
	}
	if strings.TrimSpace(pane.ID) == "" {
		return false
	}
	return true
}
