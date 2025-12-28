package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestTerminalKeyAndPaneInputCmds(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}

	cmd := m.handleTerminalKeyCmd(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if cmd == nil {
		t.Fatalf("expected terminal key cmd for scrollback key")
	}

	cmd = m.sendPaneInputCmd([]byte("hi"), "send")
	if cmd == nil {
		t.Fatalf("expected pane input cmd when client present")
	}
}
