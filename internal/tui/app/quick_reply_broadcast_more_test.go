package app

import (
	"errors"
	"strings"
	"testing"
)

func TestUniqueQuickReplyTargets(t *testing.T) {
	panes := []PaneItem{
		{ID: "p1"},
		{ID: "p1"},
		{ID: ""},
		{ID: "p2"},
	}
	targets := uniqueQuickReplyTargets(panes)
	if len(targets) != 2 {
		t.Fatalf("targets = %#v", targets)
	}
	if targets[0].Pane.ID != "p1" || targets[1].Pane.ID != "p2" {
		t.Fatalf("unexpected targets order: %#v", targets)
	}
}

func TestQuickReplyBroadcastTargets(t *testing.T) {
	m := newTestModelLite()
	session := m.selectedSession()
	session.Panes = append(session.Panes, session.Panes[0])

	targets, label := m.quickReplyTargetsForSession()
	if label != "session alpha-1" || len(targets) != 2 {
		t.Fatalf("session targets: label=%q targets=%d", label, len(targets))
	}

	targets, label = m.quickReplyTargetsForProject()
	if label != "project Alpha" || len(targets) != 3 {
		t.Fatalf("project targets: label=%q targets=%d", label, len(targets))
	}

	targets, label = m.quickReplyTargetsForAll()
	if label != "all panes" || len(targets) != 4 {
		t.Fatalf("all targets: label=%q targets=%d", label, len(targets))
	}
}

func TestQuickReplySendDetails(t *testing.T) {
	result := quickReplySendResult{
		Failed:        2,
		ClosedPaneIDs: []string{"p1"},
		Skipped:       1,
		FirstError:    "boom",
	}
	got := quickReplySendDetails(result)
	want := "2 failed, 1 closed, 1 skipped, error: boom"
	if got != want {
		t.Fatalf("details = %q, want %q", got, want)
	}
}

func TestQuickReplySendSummary(t *testing.T) {
	msg, level := quickReplySendSummary(quickReplySendResult{})
	if msg != "No panes to send to" || level != toastInfo {
		t.Fatalf("summary = %q level=%v", msg, level)
	}

	msg, level = quickReplySendSummary(quickReplySendResult{
		ScopeLabel: "project Alpha",
		Total:      3,
		Failed:     1,
		FirstError: "boom",
	})
	if level != toastWarning || !strings.Contains(msg, "No panes accepted input") {
		t.Fatalf("summary = %q level=%v", msg, level)
	}

	msg, level = quickReplySendSummary(quickReplySendResult{
		Total: 2,
		Sent:  2,
	})
	if level != toastSuccess || !strings.Contains(msg, "Sent to 2 panes") {
		t.Fatalf("summary = %q level=%v", msg, level)
	}
}

func TestApplyQuickReplyTargetResult(t *testing.T) {
	result := quickReplySendResult{}
	applyQuickReplyTargetResult(&result, quickReplyTargetResult{Status: quickReplyTargetSent})
	applyQuickReplyTargetResult(&result, quickReplyTargetResult{Status: quickReplyTargetSkipped})
	applyQuickReplyTargetResult(&result, quickReplyTargetResult{Status: quickReplyTargetClosed, PaneID: "p1"})
	applyQuickReplyTargetResult(&result, quickReplyTargetResult{Status: quickReplyTargetFailed, Err: errors.New("boom")})

	if result.Sent != 1 || result.Skipped != 1 || result.Failed != 1 || len(result.ClosedPaneIDs) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.FirstError != "boom" {
		t.Fatalf("first error = %q", result.FirstError)
	}
}

func TestQuickReplyResultForSendError(t *testing.T) {
	closed := quickReplyResultForSendError("p1", errors.New("pane closed"))
	if closed.Status != quickReplyTargetClosed || closed.PaneID != "p1" {
		t.Fatalf("closed result = %#v", closed)
	}
	failed := quickReplyResultForSendError("p2", errors.New("boom"))
	if failed.Status != quickReplyTargetFailed || failed.PaneID != "p2" {
		t.Fatalf("failed result = %#v", failed)
	}
}

func TestHandleQuickReplySend(t *testing.T) {
	m := newTestModelLite()
	msg := quickReplySendMsg{
		Result: quickReplySendResult{
			Total:         2,
			Sent:          1,
			ClosedPaneIDs: []string{"p1"},
		},
	}
	m.handleQuickReplySend(msg)
	if !m.isPaneInputDisabled("p1") {
		t.Fatalf("expected pane input disabled")
	}
	if m.toast.Text == "" {
		t.Fatalf("expected toast set")
	}
}
