package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestDashboardActionsAndNav(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}
	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}

	m.filterActive = true
	m.updateDashboard(tea.KeyMsg{Type: tea.KeyEnter})
	if m.filterActive {
		t.Fatalf("expected filter to exit on enter")
	}

	m.hardRaw = false
	m.updateDashboard(keyRune('t'))
	if !m.hardRaw {
		t.Fatalf("expected hard raw enabled")
	}

	m.updateDashboard(tea.KeyMsg{Type: tea.KeyEsc})
	if !m.hardRaw {
		t.Fatalf("expected hard raw to remain enabled")
	}
	m.updateDashboard(keyRune('t'))
	if m.hardRaw {
		t.Fatalf("expected hard raw disabled via toggle")
	}

	m.tab = TabProject
	m.selection.ProjectID = projectKey("/alpha", "Alpha")
	m.updateDashboard(keyRune('l'))
	if m.selection.ProjectID != projectKey("/beta", "Beta") {
		t.Fatalf("expected project navigation to Beta, got %#v", m.selection)
	}

	prev := m.selectionVersion
	m.updateDashboard(keyRune('j'))
	if m.selectionVersion == prev {
		t.Fatalf("expected session nav to update selection")
	}

	m.updateDashboard(keyRune('n'))
	if m.selection.Pane == "" {
		t.Fatalf("expected pane navigation to set pane")
	}

	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}
	before := m.expandedSessions["alpha-1"]
	m.updateDashboard(keyRune('g'))
	if m.expandedSessions["alpha-1"] == before {
		t.Fatalf("expected toggle panes to flip state")
	}

	m.updateDashboard(keyRune('?'))
	if m.state != StateHelp {
		t.Fatalf("expected help state")
	}

	m.setState(StateDashboard)
	_, cmd := m.updateDashboard(keyRune('q'))
	if cmd != nil {
		t.Fatalf("expected quit prompt without cmd")
	}
	if m.state != StateConfirmQuit {
		t.Fatalf("expected confirm quit state")
	}
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}
