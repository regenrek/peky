package app

import "testing"

func TestApplySessionFilter(t *testing.T) {
	m := newTestModelLite()
	m.filterInput.Blur()
	m.quickReplyInput.Focus()

	m.applySessionFilter("   ")
	if !m.filterActive || !m.filterInput.Focused() {
		t.Fatalf("expected filter active and focused")
	}
	if m.quickReplyInput.Focused() {
		t.Fatalf("expected quick reply blurred")
	}

	m.applySessionFilter("alpha")
	if m.filterActive {
		t.Fatalf("expected filter inactive")
	}
	if got := m.filterInput.Value(); got != "alpha" {
		t.Fatalf("filter value = %q", got)
	}
	if m.filterInput.Focused() || m.quickReplyInput.Focused() {
		t.Fatalf("expected inputs blurred")
	}
}

func TestPrepareQuickReplyInput(t *testing.T) {
	m := newTestModelLite()
	m.filterActive = true
	m.filterInput.Focus()
	m.quickReplyInput.Blur()

	m.prepareQuickReplyInput()
	if m.filterActive {
		t.Fatalf("expected filter inactive")
	}
	if !m.quickReplyInput.Focused() {
		t.Fatalf("expected quick reply focused")
	}
}

func TestPrefillQuickReplyInput(t *testing.T) {
	m := newTestModelLite()
	m.prefillQuickReplyInput("  hi  ")
	if got := m.quickReplyInput.Value(); got != "  hi  " {
		t.Fatalf("quick reply value = %q", got)
	}
	if !m.quickReplyInput.Focused() {
		t.Fatalf("expected quick reply focused")
	}
}
