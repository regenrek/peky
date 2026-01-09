package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuickReplyIgnoresSGRMouseJunkKeys(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.quickReplyInput.Focus()

	m.quickReplyInput.SetValue("")
	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[<65;83;19M")})
	if got := m.quickReplyInput.Value(); got != "" {
		t.Fatalf("quickReplyInput=%q want empty", got)
	}

	m.updateDashboard(keyRune('a'))
	if got := m.quickReplyInput.Value(); got != "a" {
		t.Fatalf("quickReplyInput=%q want %q", got, "a")
	}
}
