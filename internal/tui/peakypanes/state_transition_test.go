package peakypanes

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateRefreshFailureSetsToast(t *testing.T) {
	m := newTestModel(t)
	m.beginRefresh()

	_, _ = m.Update(dashboardSnapshotMsg{Result: dashboardSnapshotResult{
		Err:      errors.New("boom"),
		Settings: m.settings,
		Version:  m.selectionVersion,
	}})

	if m.refreshing || m.refreshInFlight != 0 {
		t.Fatalf("refresh flags not cleared: refreshing=%v inflight=%d", m.refreshing, m.refreshInFlight)
	}
	if m.toast.Level != toastError {
		t.Fatalf("toast level = %v, want %v", m.toast.Level, toastError)
	}
	if !strings.Contains(m.toast.Text, "Refresh failed: boom") {
		t.Fatalf("toast text = %q", m.toast.Text)
	}
}

func TestViewWithEmptyDataAndResize(t *testing.T) {
	m := newTestModel(t)

	m.state = StateHelp
	_, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	if m.state != StateHelp {
		t.Fatalf("state = %v, want %v", m.state, StateHelp)
	}
	if view := strings.TrimSpace(m.View()); view == "" {
		t.Fatalf("View(StateHelp) empty after resize")
	}

	m.state = StateDashboard
	_, _ = m.Update(dashboardSnapshotMsg{Result: dashboardSnapshotResult{
		Data:      DashboardData{},
		Settings:  m.settings,
		RawConfig: m.config,
		Version:   m.selectionVersion,
	}})
	if view := strings.TrimSpace(m.View()); view == "" {
		t.Fatalf("View(StateDashboard) empty with empty data")
	}
}
