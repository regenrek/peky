package app

import "testing"

func TestQuickReplyHistoryNavigation(t *testing.T) {
	m := newTestModelLite()
	m.quickReplyInput.SetValue("")

	m.rememberQuickReply("one")
	m.rememberQuickReply("two")
	m.rememberQuickReply("three")

	if !m.moveQuickReplyHistory(-1) {
		t.Fatalf("expected history move up to handle")
	}
	if got := m.quickReplyInput.Value(); got != "three" {
		t.Fatalf("history up = %q want %q", got, "three")
	}
	if !m.moveQuickReplyHistory(-1) {
		t.Fatalf("expected history move up to handle")
	}
	if got := m.quickReplyInput.Value(); got != "two" {
		t.Fatalf("history up = %q want %q", got, "two")
	}
	m.moveQuickReplyHistory(-1)
	if got := m.quickReplyInput.Value(); got != "one" {
		t.Fatalf("history up = %q want %q", got, "one")
	}
	m.moveQuickReplyHistory(-1)
	if got := m.quickReplyInput.Value(); got != "one" {
		t.Fatalf("history clamp = %q want %q", got, "one")
	}

	m.moveQuickReplyHistory(1)
	if got := m.quickReplyInput.Value(); got != "two" {
		t.Fatalf("history down = %q want %q", got, "two")
	}
	m.moveQuickReplyHistory(1)
	if got := m.quickReplyInput.Value(); got != "three" {
		t.Fatalf("history down = %q want %q", got, "three")
	}
	m.moveQuickReplyHistory(1)
	if got := m.quickReplyInput.Value(); got != "" {
		t.Fatalf("history exit draft = %q want empty", got)
	}
	if m.quickReplyHistoryActive() {
		t.Fatalf("expected history inactive after exit")
	}
}
