package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestDashboardNavBranches(t *testing.T) {
	m := newTestModelLite()
	m.tab = TabDashboard

	if _, handled := m.handleSessionNav(keyRune('k')); !handled {
		t.Fatalf("expected session up handled on dashboard tab")
	}
	if _, handled := m.handleSessionNav(keyRune('j')); !handled {
		t.Fatalf("expected session down handled on dashboard tab")
	}
	if _, handled := m.handleSessionNav(keyRune('K')); !handled {
		t.Fatalf("expected session only up handled on dashboard tab")
	}
	if _, handled := m.handleSessionNav(keyRune('J')); !handled {
		t.Fatalf("expected session only down handled on dashboard tab")
	}

	if _, handled := m.handlePaneNav(keyRune('n')); !handled {
		t.Fatalf("expected pane next handled on dashboard tab")
	}
	if _, handled := m.handlePaneNav(keyRune('p')); !handled {
		t.Fatalf("expected pane prev handled on dashboard tab")
	}

	m.tab = TabProject
	if _, handled := m.handleSessionNav(keyRune('k')); !handled {
		t.Fatalf("expected session up handled on project tab")
	}
	if _, handled := m.handleSessionNav(keyRune('j')); !handled {
		t.Fatalf("expected session down handled on project tab")
	}
	if _, handled := m.handleSessionNav(keyRune('K')); !handled {
		t.Fatalf("expected session only up handled on project tab")
	}
	if _, handled := m.handleSessionNav(keyRune('J')); !handled {
		t.Fatalf("expected session only down handled on project tab")
	}

	if _, handled := m.handlePaneNav(keyRune('n')); !handled {
		t.Fatalf("expected pane next handled on project tab")
	}
	if _, handled := m.handlePaneNav(keyRune('p')); !handled {
		t.Fatalf("expected pane prev handled on project tab")
	}
}

func TestTerminalFocusInput(t *testing.T) {
	m := newTestModelLite()
	m.client = &sessiond.Client{}
	m.setTerminalFocus(true)

	if cmd, handled := m.handleTerminalFocusInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}); !handled || cmd == nil {
		t.Fatalf("expected terminal input to be handled")
	}
	if cmd, handled := m.handleTerminalFocusInput(tea.KeyMsg{Type: tea.KeyEsc}); !handled || cmd == nil {
		t.Fatalf("expected terminal focus escape handled")
	}
	if m.terminalFocus {
		t.Fatalf("expected terminal focus disabled")
	}
}
