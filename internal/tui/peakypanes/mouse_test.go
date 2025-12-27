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

func seedMouseTestData(m *Model) {
	m.tab = TabDashboard
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Path: m.configPath,
		Sessions: []SessionItem{{
			Name:         "sess",
			Status:       StatusRunning,
			ActiveWindow: "1",
			Windows: []WindowItem{{
				Index: "1",
				Name:  "win",
				Panes: []PaneItem{{
					ID:    "pane-1",
					Index: "1",
					Title: "pane",
				}},
			}},
		}},
	}}}
	m.selection = selectionState{}
}
