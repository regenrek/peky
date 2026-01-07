package app

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestDashboardActionKeys(t *testing.T) {
	m := newTestModelLite()

	m.updateDashboard(keyRune('s'))
	if m.state != StateLayoutPicker {
		t.Fatalf("expected layout picker state")
	}

	m.setState(StateDashboard)
	m.updateDashboard(keyRune('o'))
	if m.state != StateProjectPicker {
		t.Fatalf("expected project picker state")
	}

	m.setState(StateDashboard)
	m.updateDashboard(keyRune('c'))
	if m.state != StateCommandPalette {
		t.Fatalf("expected command palette state")
	}

	m.setState(StateDashboard)
	m.updateDashboard(keyRune('R'))
	if m.toast.Text == "" {
		t.Fatalf("expected refresh toast")
	}

	m.setState(StateDashboard)
	m.configPath = filepath.Join(t.TempDir(), "config.yml")
	_, cmd := m.updateDashboard(keyRune('e'))
	if cmd == nil {
		t.Fatalf("expected edit config cmd")
	}

	m.setState(StateDashboard)
	m.updateDashboard(keyRune('x'))
	if m.state != StateConfirmKill {
		t.Fatalf("expected confirm kill state")
	}

	m.setState(StateDashboard)
	m.updateDashboard(keyRune('z'))
	if m.state != StateConfirmCloseProject {
		t.Fatalf("expected confirm close project state")
	}

	m.setState(StateDashboard)
	m.updateDashboard(keyRune('f'))
	if !m.filterActive {
		t.Fatalf("expected filter active")
	}

	// terminal focus input path
	m.client = &sessiond.Client{}
	m.setState(StateDashboard)
	m.setTerminalFocus(true)
	m.updateDashboard(tea.KeyMsg{Type: tea.KeyEsc})
	if !m.terminalFocus {
		t.Fatalf("expected terminal focus to remain enabled")
	}
}
