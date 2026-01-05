package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) toggleQuickReplyMode() {
	if m == nil {
		return
	}
	if m.quickReplyMode == quickReplyModePeky {
		m.setQuickReplyMode(quickReplyModePane)
		return
	}
	m.setQuickReplyMode(quickReplyModePeky)
}

func (m *Model) setQuickReplyMode(mode quickReplyMode) {
	if m == nil {
		return
	}
	if m.quickReplyMode == mode {
		return
	}
	m.quickReplyMode = mode
	m.resetQuickReplyMenu()
	m.resetQuickReplyHistory()
	m.quickReplyInput.CursorEnd()
}

func (m *Model) quickReplyModeLabel() string {
	if m == nil {
		return ""
	}
	switch m.quickReplyMode {
	case quickReplyModePeky:
		if m.pekyBusy {
			return "agent*"
		}
		return "agent"
	default:
		return "quick reply"
	}
}

func (m *Model) handlePekyToggleCommand(text string) (bool, tea.Cmd) {
	handled, target := parsePekyToggleCommand(text)
	if !handled {
		return false, nil
	}
	if target == nil {
		m.toggleQuickReplyMode()
	} else {
		m.setQuickReplyMode(*target)
	}
	m.resetQuickReplyInputState()
	if m.quickReplyMode == quickReplyModePeky {
		return true, NewInfoCmd("Peky enabled")
	}
	return true, NewInfoCmd("Peky disabled")
}

func parsePekyToggleCommand(text string) (bool, *quickReplyMode) {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	if trimmed == "/peky" {
		return true, nil
	}
	if strings.HasPrefix(trimmed, "/peky ") {
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "/peky"))
		switch rest {
		case "on", "enable", "enabled":
			mode := quickReplyModePeky
			return true, &mode
		case "off", "disable", "disabled":
			mode := quickReplyModePane
			return true, &mode
		}
		return true, nil
	}
	return false, nil
}
