package app

import (
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
)

func TestQuickReplyBroadcastSummary(t *testing.T) {
	msg, level := quickReplySendSummary(quickReplySendResult{})
	if msg == "" || level != toastInfo {
		t.Fatalf("expected empty summary info")
	}

	result := quickReplySendResult{ScopeLabel: "session s1", Total: 2, Sent: 1, Failed: 1, FirstError: "boom"}
	msg, level = quickReplySendSummary(result)
	if msg == "" || level != toastWarning {
		t.Fatalf("expected warning summary")
	}
}

func TestQuickReplyBroadcastNoClient(t *testing.T) {
	m := newTestModelLite()
	m.client = nil
	cmd := m.sendQuickReplyBroadcast(quickReplyScopeSession, "hi")
	if cmd == nil {
		t.Fatalf("expected command for missing client")
	}
	if msg := cmd(); msg == nil {
		t.Fatalf("expected message")
	}
	cmd = m.sendQuickReplyBroadcast(quickReplyScopeSession, " ")
	if cmd == nil {
		t.Fatalf("expected command for empty input")
	}
	if msg := cmd(); msg == nil {
		t.Fatalf("expected message")
	}
}

func TestQuickReplyBroadcastSendWithClient(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "broadcast")
	if len(snap.Panes) == 0 {
		t.Fatalf("expected panes")
	}
	settings, err := defaultDashboardConfig(layout.DashboardConfig{})
	if err != nil {
		t.Fatalf("defaultDashboardConfig error: %v", err)
	}
	result := buildDashboardData(dashboardSnapshotInput{
		Selection: selectionState{Session: snap.Name, Pane: snap.Panes[0].Index},
		Tab:       TabProject,
		Config:    &layout.Config{Projects: []layout.ProjectConfig{{Name: "App", Session: snap.Name, Path: snap.Path}}},
		Settings:  settings,
		Version:   1,
		Sessions:  []native.SessionSnapshot{snap},
	})
	m.data = result.Data
	m.selection = result.Resolved

	cmd := m.sendQuickReplyBroadcast(quickReplyScopeSession, "hello")
	if cmd == nil {
		t.Fatalf("expected send cmd")
	}
	msg, ok := cmd().(quickReplySendMsg)
	if !ok {
		t.Fatalf("expected quickReplySendMsg")
	}
	if msg.Result.Total == 0 {
		t.Fatalf("expected targets")
	}
	m.handleQuickReplySend(msg)
}
