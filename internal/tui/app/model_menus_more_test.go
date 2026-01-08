package app

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestUpdateSettingsMenuEscReturnsDashboard(t *testing.T) {
	m := newTestModelLite()
	_ = m.openSettingsMenu()
	if m.state != StateSettingsMenu {
		t.Fatalf("state=%v want=%v", m.state, StateSettingsMenu)
	}
	_, _ = m.updateSettingsMenu(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("state=%v want=%v", m.state, StateDashboard)
	}
}

func TestUpdateSettingsMenuEnterRunsSelection(t *testing.T) {
	m := newTestModelLite()
	_ = m.openSettingsMenu()
	_, _ = m.updateSettingsMenu(tea.KeyMsg{Type: tea.KeyDown})
	_, _ = m.updateSettingsMenu(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StatePerformanceMenu {
		t.Fatalf("state=%v want=%v", m.state, StatePerformanceMenu)
	}
}

func TestUpdatePerformanceMenuEscReturnsToSettings(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := layout.SaveConfig(cfgPath, &layout.Config{}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := newTestModelLite()
	m.configPath = cfgPath
	_ = m.openPerformanceMenu()
	if m.state != StatePerformanceMenu {
		t.Fatalf("state=%v want=%v", m.state, StatePerformanceMenu)
	}
	_, _ = m.updatePerformanceMenu(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateSettingsMenu {
		t.Fatalf("state=%v want=%v", m.state, StateSettingsMenu)
	}
}

func TestUpdateDebugMenuEnterRunsRestartConfirm(t *testing.T) {
	m := newTestModelLite()
	_ = m.openDebugMenu()
	if m.state != StateDebugMenu {
		t.Fatalf("state=%v want=%v", m.state, StateDebugMenu)
	}
	_, _ = m.updateDebugMenu(tea.KeyMsg{Type: tea.KeyDown})
	_, _ = m.updateDebugMenu(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateConfirmRestart {
		t.Fatalf("state=%v want=%v", m.state, StateConfirmRestart)
	}
}

func TestRunPerformanceMenuSelectionRefreshesItems(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := layout.SaveConfig(cfgPath, &layout.Config{}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := newTestModelLite()
	m.configPath = cfgPath
	_ = m.openPerformanceMenu()
	if m.state != StatePerformanceMenu {
		t.Fatalf("state=%v want=%v", m.state, StatePerformanceMenu)
	}
	m.perfMenu.Select(0)
	if cmd := m.runPerformanceMenuSelection(); cmd == nil {
		t.Fatalf("expected refresh cmd")
	}
	if m.state != StatePerformanceMenu {
		t.Fatalf("state=%v want=%v", m.state, StatePerformanceMenu)
	}
}

func TestRunPerformanceMenuSelectionBackReturnsToSettings(t *testing.T) {
	m := newTestModelLite()
	_ = m.openPerformanceMenu()
	m.perfMenu.Select(4)
	_ = m.runPerformanceMenuSelection()
	if m.state != StateSettingsMenu {
		t.Fatalf("state=%v want=%v", m.state, StateSettingsMenu)
	}
}
