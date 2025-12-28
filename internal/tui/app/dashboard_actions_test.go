package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestDashboardActionsAndNav(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}
	m.selection = selectionState{Project: "Alpha", Session: "alpha-1", Pane: "1"}

	m.filterActive = true
	m.updateDashboard(tea.KeyMsg{Type: tea.KeyEnter})
	if m.filterActive {
		t.Fatalf("expected filter to exit on enter")
	}

	m.terminalFocus = false
	m.updateDashboard(keyRune('t'))
	if !m.terminalFocus {
		t.Fatalf("expected terminal focus enabled")
	}

	m.updateDashboard(tea.KeyMsg{Type: tea.KeyEsc})
	if !m.terminalFocus {
		t.Fatalf("expected terminal focus to remain enabled")
	}
	m.updateDashboard(keyRune('t'))
	if m.terminalFocus {
		t.Fatalf("expected terminal focus disabled via toggle")
	}

	m.tab = TabProject
	m.selection.Project = "Alpha"
	m.updateDashboard(keyRune('l'))
	if m.selection.Project != "Beta" {
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

	m.selection = selectionState{Project: "Alpha", Session: "alpha-1", Pane: "1"}
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
	if cmd == nil {
		t.Fatalf("expected quit cmd")
	}
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}
