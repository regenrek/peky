package app

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestUpdateHandlers(t *testing.T) {
	m := newTestModelLite()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if m.width != 100 || m.height != 40 {
		t.Fatalf("expected window size applied")
	}

	m.Update(SuccessMsg{Message: "ok"})
	if m.toast.Text == "" {
		t.Fatalf("expected success toast")
	}
	m.Update(ErrorMsg{Err: errors.New("boom")})
	if m.toast.Text == "" {
		t.Fatalf("expected error toast")
	}

	m.refreshInFlight = 0
	m.handleRefreshTick(refreshTickMsg{})
	if m.refreshInFlight == 0 {
		t.Fatalf("expected refresh in flight")
	}

	version := m.selectionVersion
	m.handleSelectionRefresh(selectionRefreshMsg{Version: version})

	snapshot := dashboardSnapshotMsg{Result: dashboardSnapshotResult{
		Data:     DashboardData{Projects: sampleProjects()},
		Settings: m.settings,
		RawConfig: &layout.Config{
			Dashboard: layout.DashboardConfig{},
		},
		Version: version,
	}}
	m.handleDashboardSnapshot(snapshot)
	if len(m.data.Projects) == 0 {
		t.Fatalf("expected dashboard data applied")
	}

	view := sessiond.PaneViewResponse{PaneID: "p1", Cols: 1, Rows: 1, View: "view", AllowMotion: true}
	m.handlePaneViews(paneViewsMsg{Views: []sessiond.PaneViewResponse{view}})
	key := paneViewKeyFrom(view)
	if m.paneViews[key] != "view" {
		t.Fatalf("expected pane view stored")
	}

	m.handleSessionStarted(sessionStartedMsg{Name: "alpha-1", Path: "/alpha", Focus: true})
	if m.selection.Session != "alpha-1" {
		t.Fatalf("expected selection updated")
	}
}

func TestHandleDashboardSnapshotRefreshSeq(t *testing.T) {
	m := newTestModelLite()
	m.data = DashboardData{Projects: []ProjectGroup{{Name: "Keep"}}}
	m.refreshSeq = 5
	m.lastAppliedSeq = 4

	stale := dashboardSnapshotMsg{Result: dashboardSnapshotResult{
		Data:       DashboardData{Projects: []ProjectGroup{{Name: "Stale"}}},
		Settings:   m.settings,
		Version:    m.selectionVersion,
		RefreshSeq: 4,
	}}
	m.handleDashboardSnapshot(stale)
	if m.data.Projects[0].Name != "Keep" {
		t.Fatalf("stale refresh applied: %#v", m.data.Projects)
	}

	fresh := dashboardSnapshotMsg{Result: dashboardSnapshotResult{
		Data:       DashboardData{Projects: []ProjectGroup{{Name: "Fresh"}}},
		Settings:   m.settings,
		Version:    m.selectionVersion,
		RefreshSeq: 5,
	}}
	m.handleDashboardSnapshot(fresh)
	if m.data.Projects[0].Name != "Fresh" {
		t.Fatalf("fresh refresh not applied: %#v", m.data.Projects)
	}
	if m.lastAppliedSeq != 5 {
		t.Fatalf("lastAppliedSeq=%d want 5", m.lastAppliedSeq)
	}
}

func TestUpdateRouting(t *testing.T) {
	m := newTestModelLite()

	m.state = StateHelp
	if _, _, handled := m.handleMouseMsg(tea.MouseMsg{}); handled {
		t.Fatalf("expected mouse msg ignored outside dashboard")
	}

	m.state = StateDashboard
	if _, _, handled := m.handleMouseMsg(tea.MouseMsg{}); !handled {
		t.Fatalf("expected mouse msg handled in dashboard")
	}

	m.state = StateHelp
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected key msg handled")
	}

	m.state = StateProjectPicker
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}); !handled {
		t.Fatalf("expected project picker handled")
	}

	m.state = StateProjectPicker
	if _, _, handled := m.handlePickerUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}); !handled {
		t.Fatalf("expected picker update handled")
	}

	m.filterActive = true
	if _, _, handled := m.handlePassiveUpdates(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}); !handled {
		t.Fatalf("expected passive update handled")
	}
}

func TestHandlePickerUpdateMoreStates(t *testing.T) {
	m := newTestModelLite()

	states := []ViewState{
		StateLayoutPicker,
		StatePaneSwapPicker,
		StateCommandPalette,
	}
	for _, state := range states {
		m.state = state
		if _, _, handled := m.handlePickerUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}); !handled {
			t.Fatalf("expected picker update handled for %v", state)
		}
	}
}
