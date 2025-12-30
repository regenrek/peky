package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const quickReplyHistoryMax = 20

func (m *Model) resetQuickReplyHistory() {
	m.quickReplyHistoryIndex = -1
	m.quickReplyHistoryDraft = ""
}

func (m *Model) rememberQuickReply(value string) {
	text := strings.TrimSpace(value)
	if text == "" {
		return
	}
	if len(m.quickReplyHistory) > 0 && m.quickReplyHistory[len(m.quickReplyHistory)-1] == text {
		return
	}
	m.quickReplyHistory = append(m.quickReplyHistory, text)
	if len(m.quickReplyHistory) > quickReplyHistoryMax {
		excess := len(m.quickReplyHistory) - quickReplyHistoryMax
		m.quickReplyHistory = m.quickReplyHistory[excess:]
	}
}

func (m *Model) quickReplyHistoryActive() bool {
	return m.quickReplyHistoryIndex >= 0
}

func (m *Model) moveQuickReplyHistory(delta int) bool {
	if len(m.quickReplyHistory) == 0 {
		return false
	}
	if m.quickReplyHistoryIndex == -1 {
		m.quickReplyHistoryDraft = m.quickReplyInput.Value()
		if delta < 0 {
			m.quickReplyHistoryIndex = len(m.quickReplyHistory) - 1
		} else {
			return false
		}
	} else {
		next := m.quickReplyHistoryIndex + delta
		if next < 0 {
			next = 0
		}
		if next >= len(m.quickReplyHistory) {
			m.quickReplyHistoryIndex = -1
			m.quickReplyInput.SetValue(m.quickReplyHistoryDraft)
			m.quickReplyInput.CursorEnd()
			m.quickReplyHistoryDraft = ""
			return true
		}
		m.quickReplyHistoryIndex = next
	}
	if m.quickReplyHistoryIndex >= 0 {
		m.quickReplyInput.SetValue(m.quickReplyHistory[m.quickReplyHistoryIndex])
		m.quickReplyInput.CursorEnd()
	}
	return true
}

func shouldExitQuickReplyHistory(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes, tea.KeyBackspace, tea.KeyDelete:
		return true
	default:
		return false
	}
}
