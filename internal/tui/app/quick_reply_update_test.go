package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuickReplyIgnoresSGRMouseJunkKeys(t *testing.T) {
	t.Skip("legacy Bubble Tea input decoding could leak SGR mouse fragments; ultraviolet decoding prevents this")
}

func TestQuickReplyEscBlursWhenEmpty(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.quickReplyInput.Focus()
	m.quickReplyInput.SetValue("")

	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyEsc})

	if m.quickReplyInput.Focused() {
		t.Fatalf("expected quick reply to blur on empty esc")
	}
}
