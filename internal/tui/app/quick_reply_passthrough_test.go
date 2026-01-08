package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestQuickReplyPassthroughUpDoesNotActivateHistory(t *testing.T) {
	m := newTestModelLite()
	m.rememberQuickReply("one")
	m.quickReplyInput.SetValue("")

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyUp})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if _, ok := cmd().(ErrorMsg); !ok {
		t.Fatalf("expected ErrorMsg when client nil")
	}
	if m.quickReplyHistoryActive() {
		t.Fatalf("expected quick reply history inactive")
	}
	if got := m.quickReplyInput.Value(); got != "" {
		t.Fatalf("expected quick reply input unchanged, got %q", got)
	}
}

func TestQuickReplyPassthroughEnterDoesNotAttachOrStart(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("")

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if _, ok := cmd().(ErrorMsg); !ok {
		t.Fatalf("expected ErrorMsg when client nil")
	}
	if got := m.quickReplyInput.Value(); got != "" {
		t.Fatalf("expected quick reply input unchanged, got %q", got)
	}
}

func TestQuickReplyHistoryStillWorksWhenTyping(t *testing.T) {
	m := newTestModelLite()
	m.rememberQuickReply("one")
	m.quickReplyInput.SetValue("draft")

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Fatalf("expected no cmd for history navigation")
	}
	if !m.quickReplyHistoryActive() {
		t.Fatalf("expected quick reply history active")
	}
	if got := m.quickReplyInput.Value(); got != "one" {
		t.Fatalf("expected history value %q, got %q", "one", got)
	}
}

func TestQuickReplyPassthroughCtrlLClearsPane(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("")

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyCtrlL})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if _, ok := cmd().(ErrorMsg); !ok {
		t.Fatalf("expected ErrorMsg when client nil")
	}
}

func TestQuickReplyPassthroughDoesNotStealPaneNavKeys(t *testing.T) {
	m := newTestModelLite()
	km, err := buildDashboardKeyMap(layout.DashboardKeymapConfig{})
	if err != nil {
		t.Fatalf("buildDashboardKeyMap: %v", err)
	}
	m.keys = km
	m.quickReplyInput.SetValue("")

	before := m.selection.Pane
	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyCtrlD})
	if cmd != nil {
		t.Fatalf("expected no cmd for pane nav")
	}
	if before == m.selection.Pane {
		t.Fatalf("expected pane selection to change")
	}
}

func TestQuickReplyPekyModeKeepsEmptyEnterInfo(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyMode = quickReplyModePeky
	m.quickReplyInput.SetValue("")

	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if _, ok := cmd().(InfoMsg); !ok {
		t.Fatalf("expected InfoMsg")
	}
}
