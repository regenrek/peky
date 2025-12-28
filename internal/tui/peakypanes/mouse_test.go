package peakypanes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

	if m.selection != hit.Selection {
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

func TestEscExitsTerminalFocus(t *testing.T) {
	m := newTestModel(t)
	seedMouseTestData(m)
	m.setTerminalFocus(true)

	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyEsc})
	if m.terminalFocus {
		t.Fatalf("terminalFocus should be false after esc")
	}
}

func TestMouseHeaderClickSwitchesToProjectTab(t *testing.T) {
	m := newTestModel(t)
	seedHeaderTestData(m)

	rect, ok := findHeaderRect(t, m, headerPartProject, "Beta")
	if !ok {
		t.Fatalf("project header rect not found")
	}
	msg := tea.MouseMsg{X: rect.X + 1, Y: rect.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	_, _ = m.updateDashboardMouse(msg)

	if m.tab != TabProject {
		t.Fatalf("tab=%v want %v", m.tab, TabProject)
	}
	if m.selection.Project != "Beta" {
		t.Fatalf("selection project=%q want %q", m.selection.Project, "Beta")
	}
}

func TestMouseHeaderClickSwitchesToDashboardTab(t *testing.T) {
	m := newTestModel(t)
	seedHeaderTestData(m)
	m.tab = TabProject
	m.selection.Project = "Beta"

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
	m.selection = selectionState{Project: "Alpha", Session: "alpha", Pane: "1"}
}

func findHeaderRect(t *testing.T, m *Model, kind headerPartKind, project string) (rect, bool) {
	t.Helper()
	for _, hit := range m.headerHitRects() {
		if hit.Hit.Kind != kind {
			continue
		}
		if kind == headerPartProject && hit.Hit.ProjectName != project {
			continue
		}
		return hit.Rect, true
	}
	return rect{}, false
}
