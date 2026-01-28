package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	tuiinput "github.com/regenrek/peakypanes/internal/tui/input"
)

func (m *Model) updateQuickReplyInput(msg tuiinput.KeyMsg) (tea.Model, tea.Cmd) {
	if m == nil || !m.quickReplyEnabled() {
		return m, nil
	}
	teaMsg := msg.Tea()
	if teaMsg.Type == tea.KeySpace {
		teaMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	}
	if !agentFeaturesEnabled && m.quickReplyMode == quickReplyModePeky {
		m.setQuickReplyMode(quickReplyModePane)
	}
	if m.applyQuickReplyCompletionOnTab(teaMsg) {
		return m, nil
	}
	if handled, cmd := m.handleQuickReplyPaneNav(msg); handled {
		return m, cmd
	}
	if m.handleQuickReplyMenuNav(teaMsg) {
		return m, nil
	}
	m.maybeExitQuickReplyHistory(teaMsg)
	if m.handleQuickReplyHistoryNav(teaMsg) {
		return m, nil
	}
	if handled, cmd := m.handleQuickReplySubmit(teaMsg); handled {
		return m, cmd
	}
	if m.handleQuickReplyEscape(teaMsg) {
		return m, nil
	}
	var cmd tea.Cmd
	m.quickReplyInput, cmd = m.quickReplyInput.Update(teaMsg)
	m.updateQuickReplyMenuSelection()
	return m, cmd
}

func (m *Model) applyQuickReplyCompletionOnTab(msg tea.KeyMsg) bool {
	return msg.String() == "tab" && m.applyQuickReplyMenuCompletion()
}

func (m *Model) handleQuickReplyPaneNav(msg tuiinput.KeyMsg) (bool, tea.Cmd) {
	switch {
	case matchesBinding(msg, m.keys.paneNext):
		return true, m.cyclePane(1)
	case matchesBinding(msg, m.keys.panePrev):
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
		return true, nil
	}
	if m.quickReplyMode == quickReplyModePeky {
		if outcome := m.handleAgentSlashCommand(text); outcome.Handled {
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
	if strings.TrimSpace(m.quickReplyInput.Value()) == "" {
		m.quickReplyMouseSel.clear()
		m.resetQuickReplyHistory()
		m.resetQuickReplyMenu()
		m.quickReplyInput.Blur()
		return true
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
