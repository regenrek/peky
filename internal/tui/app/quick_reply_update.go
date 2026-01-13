package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateQuickReply(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !agentFeaturesEnabled && m.quickReplyMode == quickReplyModePeky {
		m.setQuickReplyMode(quickReplyModePane)
	}
	streamCmd := m.maybeQueueQuickReplyStream(msg)
	if m.handleQuickReplyModeToggle(msg) {
		return m, streamCmd
	}
	if m.applyQuickReplyCompletionOnTab(msg) {
		return m, streamCmd
	}
	if handled, cmd := m.handleQuickReplyPassthrough(msg); handled {
		return m, tea.Batch(streamCmd, cmd)
	}
	if handled, cmd := m.handleQuickReplyPaneNav(msg); handled {
		return m, tea.Batch(streamCmd, cmd)
	}
	if m.handleQuickReplyMenuNav(msg) {
		return m, streamCmd
	}
	m.maybeExitQuickReplyHistory(msg)
	if m.handleQuickReplyHistoryNav(msg) {
		return m, streamCmd
	}
	if handled, cmd := m.handleQuickReplySubmit(msg); handled {
		return m, tea.Batch(streamCmd, cmd)
	}
	if m.handleQuickReplyEscape(msg) {
		return m, streamCmd
	}
	if isSGRMouseKeyJunk(msg) {
		return m, streamCmd
	}
	var cmd tea.Cmd
	m.quickReplyInput, cmd = m.quickReplyInput.Update(msg)
	m.updateQuickReplyMenuSelection()
	return m, tea.Batch(streamCmd, cmd)
}

func isSGRMouseKeyJunk(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes {
		return false
	}
	s := msg.String()
	if strings.HasPrefix(s, "[<") {
		s = strings.TrimPrefix(s, "[<")
	} else if strings.HasPrefix(s, "<") {
		s = strings.TrimPrefix(s, "<")
	} else {
		return false
	}
	if len(s) == 0 {
		return false
	}
	last := s[len(s)-1]
	if last != 'M' && last != 'm' {
		return false
	}
	body := s[:len(s)-1]
	parts := strings.Split(body, ";")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func (m *Model) handleQuickReplyPassthrough(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m == nil || m.terminalFocus || m.quickReplyMode != quickReplyModePane {
		return false, nil
	}
	if m.quickReplyHistoryActive() {
		return false, nil
	}
	if strings.TrimSpace(m.quickReplyInput.Value()) != "" {
		return false, nil
	}

	switch msg.String() {
	case "enter", "esc", "up", "down", "left", "right", "tab", "pgup", "pgdown", "home", "end", "ctrl+l":
	default:
		return false, nil
	}
	if menu := m.quickReplyMenuState(); menu.kind != quickReplyMenuNone {
		return false, nil
	}
	payload := encodeKeyMsg(msg)
	if len(payload) == 0 {
		return true, nil
	}
	return true, m.sendPaneInputCmd(payload, "quick reply passthrough")
}

func (m *Model) handleQuickReplyModeToggle(msg tea.KeyMsg) bool {
	if msg.String() != "shift+tab" {
		return false
	}
	if !agentFeaturesEnabled {
		m.setQuickReplyMode(quickReplyModePane)
		m.setToast("Agent mode disabled", toastWarning)
		return true
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
	if m.quickReplyStreamEnabled() {
		return false
	}
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
	if m.quickReplyMode == quickReplyModePane && m.quickReplyStreamEnabled() && !strings.HasPrefix(strings.TrimLeft(text, " \t"), "/") {
		paneID := m.selectedPaneID()
		m.rememberQuickReply(text)
		m.resetQuickReplyInputState()
		m.resetQuickReplyHistory()
		return true, m.flushQuickReplyStreamWithEnter(paneID)
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
	m.resetQuickReplyInputState()
	return true
}

func (m *Model) resetQuickReplyInputState() {
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.CursorEnd()
	m.resetQuickReplyHistory()
	m.resetQuickReplyMenu()
}
