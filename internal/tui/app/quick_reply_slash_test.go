package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuickReplySlashKillPane(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("/kill")

	m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.quickReplyInput.Value(); got != "/kill " {
		t.Fatalf("expected slash completion on enter, got %q", got)
	}
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state after completion, got %v", m.state)
	}

	m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateConfirmClosePane {
		t.Fatalf("expected confirm close pane state after second enter, got %v", m.state)
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

func TestQuickReplySlashSuggestionsRequirePrefix(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("hello /")

	if got := m.slashSuggestions(); len(got) != 0 {
		t.Fatalf("expected no suggestions for mid-input slash, got %#v", got)
	}
}

func TestQuickReplySlashSuggestionsOnSlash(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("")
	m.quickReplyInput.Focus()

	m.updateQuickReply(keyRune('/'))
	if got := m.quickReplyInput.Value(); got != "/" {
		t.Fatalf("expected input '/', got %q", got)
	}
	suggestions := m.slashSuggestions()
	if len(suggestions) == 0 {
		t.Fatalf("expected slash suggestions")
	}
	for _, suggestion := range suggestions {
		if suggestion.Text == "" || suggestion.Text[0] != '/' {
			t.Fatalf("unexpected suggestion %#v", suggestion)
		}
	}
}

func TestQuickReplySlashCompletionExactMatch(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("/kill")

	if !m.applySlashCompletion() {
		t.Fatalf("expected slash completion")
	}
	if got := m.quickReplyInput.Value(); got != "/kill " {
		t.Fatalf("expected /kill completion, got %q", got)
	}
}

func TestQuickReplySlashCompletionLongestPrefix(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("/ki")

	if !m.applySlashCompletion() {
		t.Fatalf("expected slash completion")
	}
	if got := m.quickReplyInput.Value(); got != "/kill " {
		t.Fatalf("expected /kill prefix completion, got %q", got)
	}
}
