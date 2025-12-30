package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuickReplySlashKillPane(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("/kill")

	m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateConfirmClosePane {
		t.Fatalf("expected confirm close pane state, got %v", m.state)
	}
}

func TestQuickReplySlashUnknownKeepsInput(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("/nope")

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected warning cmd")
	}
	if _, ok := cmd().(WarningMsg); !ok {
		t.Fatalf("expected WarningMsg for unknown slash command")
	}
	if got := m.quickReplyInput.Value(); got != "/nope" {
		t.Fatalf("expected input preserved, got %q", got)
	}
}

func TestQuickReplySlashFilterSetsValue(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("/filter alpha")

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		_ = cmd()
	}
	if got := m.filterInput.Value(); got != "alpha" {
		t.Fatalf("expected filter value %q, got %q", "alpha", got)
	}
	if m.filterActive {
		t.Fatalf("expected filter inactive when value provided")
	}
	if got := m.quickReplyInput.Value(); got != "" {
		t.Fatalf("expected quick reply cleared, got %q", got)
	}
}

func TestQuickReplyBroadcastParsesAll(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("/all reset")

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected broadcast cmd")
	}
	if _, ok := cmd().(ErrorMsg); !ok {
		t.Fatalf("expected error when client missing")
	}
	if got := m.quickReplyInput.Value(); got != "" {
		t.Fatalf("expected quick reply cleared after broadcast, got %q", got)
	}
}

func TestQuickReplySlashPaletteOpensAfterDelay(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("")

	_, cmd := m.updateQuickReply(keyRune('/'))
	if cmd == nil {
		t.Fatalf("expected slash palette cmd")
	}
	if !m.quickReplySlashPending {
		t.Fatalf("expected slash palette pending")
	}
	if got := m.quickReplyInput.Value(); got != "/" {
		t.Fatalf("expected input set to '/', got %q", got)
	}

	cmd = m.handleSlashPaletteMsg()
	if cmd == nil {
		t.Fatalf("expected palette open cmd")
	}
	if m.state != StateCommandPalette {
		t.Fatalf("expected command palette state, got %v", m.state)
	}
}
