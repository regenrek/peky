package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) renamePaneDirect(newName string) tea.Cmd {
	name := strings.TrimSpace(newName)
	if err := validateSessionName(name); err != nil {
		return NewWarningCmd(err.Error())
	}
	session := m.selectedSession()
	if session == nil {
		return NewWarningCmd("No session selected")
	}
	pane := m.selectedPane()
	if pane == nil {
		return NewWarningCmd("No pane selected")
	}
	m.renameSession = session.Name
	m.renamePane = pane.Title
	m.renamePaneIndex = pane.Index
	return m.applyRenamePane(name)
}

func (m *Model) renameSessionDirect(newName string) tea.Cmd {
	name := strings.TrimSpace(newName)
	if err := validateSessionName(name); err != nil {
		return NewWarningCmd(err.Error())
	}
	session := m.selectedSession()
	if session == nil {
		return NewWarningCmd("No session selected")
	}
	m.renameSession = session.Name
	return m.applyRenameSession(name)
}

func (m *Model) applySessionFilter(value string) tea.Cmd {
	filter := strings.TrimSpace(value)
	if filter == "" {
		m.filterActive = true
		m.filterInput.Focus()
		m.quickReplyInput.Blur()
		return nil
	}
	m.filterActive = false
	m.filterInput.SetValue(filter)
	m.filterInput.CursorEnd()
	m.filterInput.Blur()
	m.quickReplyInput.Blur()
	return nil
}

func (m *Model) prepareQuickReplyInput() tea.Cmd {
	var cmd tea.Cmd
	if m.hardRaw {
		cmd = m.setHardRaw(false)
	}
	if m.filterActive {
		m.filterActive = false
		m.filterInput.Blur()
	}
	m.quickReplyInput.Focus()
	return cmd
}

func (m *Model) prefillQuickReplyInput(value string) tea.Cmd {
	cmd := m.prepareQuickReplyInput()
	m.quickReplyInput.SetValue(value)
	m.quickReplyInput.CursorEnd()
	m.updateQuickReplyMenuSelection()
	return cmd
}
