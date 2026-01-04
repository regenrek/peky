package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateQuickReply(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.handleQuickReplyModeToggle(msg) {
		return m, nil
	}
	if m.applyQuickReplyCompletionOnTab(msg) {
		return m, nil
	}
	if handled, cmd := m.handleQuickReplyPaneNav(msg); handled {
		return m, cmd
	}
	if m.handleQuickReplyMenuNav(msg) {
		return m, nil
	}
	m.maybeExitQuickReplyHistory(msg)
	if m.handleQuickReplyHistoryNav(msg) {
		return m, nil
	}
	if handled, cmd := m.handleQuickReplySubmit(msg); handled {
		return m, cmd
	}
	if m.handleQuickReplyEscape(msg) {
		return m, nil
	}
	var cmd tea.Cmd
	m.quickReplyInput, cmd = m.quickReplyInput.Update(msg)
	m.updateQuickReplyMenuSelection()
	return m, cmd
}

func (m *Model) handleQuickReplyModeToggle(msg tea.KeyMsg) bool {
	if msg.String() != "shift+tab" {
		return false
	}
	m.toggleQuickReplyMode()
	return true
}

func (m *Model) applyQuickReplyCompletionOnTab(msg tea.KeyMsg) bool {
	return msg.String() == "tab" && m.applyQuickReplyMenuCompletion()
}

func (m *Model) handleQuickReplyPaneNav(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.paneNext):
		return true, m.cyclePane(1)
	case key.Matches(msg, m.keys.panePrev):
		return true, m.cyclePane(-1)
	default:
		return false, nil
	}
}

func (m *Model) handleQuickReplyMenuNav(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up":
		return m.moveQuickReplyMenuSelection(-1)
	case "down":
		return m.moveQuickReplyMenuSelection(1)
	default:
		return false
	}
}

func (m *Model) maybeExitQuickReplyHistory(msg tea.KeyMsg) {
	if m.quickReplyHistoryActive() && shouldExitQuickReplyHistory(msg) {
		m.resetQuickReplyHistory()
	}
}

func (m *Model) handleQuickReplyHistoryNav(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up":
		if m.moveQuickReplyHistory(-1) {
			m.updateQuickReplyMenuSelection()
			return true
		}
	case "down":
		if m.moveQuickReplyHistory(1) {
			m.updateQuickReplyMenuSelection()
			return true
		}
	}
	return false
}

func (m *Model) handleQuickReplySubmit(msg tea.KeyMsg) (bool, tea.Cmd) {
	if msg.String() != "enter" {
		return false, nil
	}
	if m.applyQuickReplyMenuCompletion() {
		return true, nil
	}
	text := strings.TrimSpace(m.quickReplyInput.Value())
	if handled, cmd := m.handlePekyToggleCommand(text); handled {
		return true, cmd
	}
	if text == "" {
		if m.quickReplyMode == quickReplyModePeky {
			return true, NewInfoCmd("Enter a prompt")
		}
		return true, m.attachOrStart()
	}
	if m.quickReplyMode == quickReplyModePeky {
		m.rememberQuickReply(text)
		m.resetQuickReplyHistory()
		return true, m.sendPekyPrompt(text)
	}
	outcome := m.handleQuickReplyCommand(text)
	if outcome.Handled {
		if outcome.RecordPrompt {
			m.rememberQuickReply(text)
		}
		if outcome.ClearInput {
			m.resetQuickReplyInputState()
		}
		return true, outcome.Cmd
	}
	m.rememberQuickReply(text)
	m.resetQuickReplyHistory()
	return true, m.sendQuickReply()
}

func (m *Model) handleQuickReplyEscape(msg tea.KeyMsg) bool {
	if msg.String() != "esc" {
		return false
	}
	m.resetQuickReplyInputState()
	return true
}

func (m *Model) resetQuickReplyInputState() {
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.CursorEnd()
	m.resetQuickReplyHistory()
	m.resetQuickReplyMenu()
}
