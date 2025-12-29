package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/tui/mouse"
)

func TestMouseSingleClickSelectsPane(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)

	hits := m.dashboardPaneHits()
	if len(hits) == 0 {
		t.Fatalf("expected dashboard pane hits")
	}
	hit := hits[0]
	msg := tea.MouseMsg{X: hit.Outer.X + 1, Y: hit.Outer.Y + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.selection != selectionFromMouse(hit.Selection) {
		t.Fatalf("selection=%#v want %#v", m.selection, hit.Selection)
	}
	if m.terminalFocus {
		t.Fatalf("terminalFocus should remain false after single click")
	}
}

func TestMouseDoubleClickEntersFocus(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)

	hits := m.dashboardPaneHits()
	if len(hits) == 0 {
		t.Fatalf("expected dashboard pane hits")
	}
	hit := hits[0]
	msg := tea.MouseMsg{X: hit.Outer.X + 1, Y: hit.Outer.Y + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)
	if m.terminalFocus {
		t.Fatalf("terminalFocus should remain false after first click")
	}

	_, _ = m.updateDashboardMouse(msg)
	if !m.terminalFocus {
		t.Fatalf("terminalFocus should be true after double click")
	}
}

func TestEscKeepsTerminalFocus(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.setTerminalFocus(true)

	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyEsc})
	if !m.terminalFocus {
		t.Fatalf("terminalFocus should remain true after esc")
	}
}

func TestMouseHeaderClickSwitchesToProjectTab(t *testing.T) {
	m := newTestModel(t)
	seedHeaderTestData(m)

	rect, ok := findHeaderRect(t, m, headerPartProject, projectKey("/tmp/beta", "Beta"))
	if !ok {
		t.Fatalf("project header rect not found")
	}
	msg := tea.MouseMsg{X: rect.X + 1, Y: rect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.tab != TabProject {
		t.Fatalf("tab=%v want %v", m.tab, TabProject)
	}
	if m.selection.ProjectID != projectKey("/tmp/beta", "Beta") {
		t.Fatalf("selection project=%q want %q", m.selection.ProjectID, projectKey("/tmp/beta", "Beta"))
	}
}

func TestMouseHeaderClickSwitchesToDashboardTab(t *testing.T) {
	m := newTestModel(t)
	seedHeaderTestData(m)
	m.tab = TabProject
	m.selection.ProjectID = projectKey("/tmp/beta", "Beta")

	rect, ok := findHeaderRect(t, m, headerPartDashboard, "")
	if !ok {
		t.Fatalf("dashboard header rect not found")
	}
	msg := tea.MouseMsg{X: rect.X + 1, Y: rect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.tab != TabDashboard {
		t.Fatalf("tab=%v want %v", m.tab, TabDashboard)
	}
}

func TestMouseHeaderClickNewOpensProjectPicker(t *testing.T) {
	m := newTestModel(t)
	seedHeaderTestData(m)
	m.settings.ProjectRoots = []string{t.TempDir()}

	rect, ok := findHeaderRect(t, m, headerPartNew, "")
	if !ok {
		t.Fatalf("new header rect not found")
	}
	msg := tea.MouseMsg{X: rect.X + 1, Y: rect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.state != StateProjectPicker {
		t.Fatalf("state=%v want %v", m.state, StateProjectPicker)
	}
}

func seedMouseTestData(m *Model) {
	m.tab = TabDashboard
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:   projectKey(m.configPath, "Proj"),
		Name: "Proj",
		Path: m.configPath,
		Sessions: []SessionItem{{
			Name:       "sess",
			Status:     StatusRunning,
			ActivePane: "1",
			Panes: []PaneItem{{
				ID:    "pane-1",
				Index: "1",
				Title: "pane",
			}},
		}},
	}}}
	m.selection = selectionState{}
}

func seedHeaderTestData(m *Model) {
	m.tab = TabDashboard
	m.data = DashboardData{Projects: []ProjectGroup{
		{
			ID:   projectKey("/tmp/alpha", "Alpha"),
			Name: "Alpha",
			Path: "/tmp/alpha",
			Sessions: []SessionItem{{
				Name:   "alpha",
				Status: StatusRunning,
				Panes: []PaneItem{{
					ID:    "pane-a",
					Index: "1",
					Title: "pane",
				}},
			}},
		},
		{
			ID:   projectKey("/tmp/beta", "Beta"),
			Name: "Beta",
			Path: "/tmp/beta",
			Sessions: []SessionItem{{
				Name:   "beta",
				Status: StatusRunning,
				Panes: []PaneItem{{
					ID:    "pane-b",
					Index: "1",
					Title: "pane",
				}},
			}},
		},
	}}
	m.selection = selectionState{ProjectID: projectKey("/tmp/alpha", "Alpha"), Session: "alpha", Pane: "1"}
}

func findHeaderRect(t *testing.T, m *Model, kind headerPartKind, projectID string) (mouse.Rect, bool) {
	t.Helper()
	wantKind, ok := headerHitKind(kind)
	if !ok {
		t.Fatalf("headerHitKind(%v) not mapped", kind)
	}
	for _, hit := range m.headerHitRects() {
		if hit.Hit.Kind != wantKind {
			continue
		}
		if wantKind == mouse.HeaderProject && hit.Hit.ProjectID != projectID {
			continue
		}
		return hit.Rect, true
	}
	return mouse.Rect{}, false
}
