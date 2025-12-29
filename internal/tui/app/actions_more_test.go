package app

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKillAndCloseFlows(t *testing.T) {
	m := newTestModelLite()
	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}

	m.openKillConfirm()
	if m.state != StateConfirmKill || m.confirmSession == "" {
		t.Fatalf("expected confirm kill state")
	}
	m.updateConfirmKill(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.state != StateDashboard || m.confirmSession != "" {
		t.Fatalf("expected cancel kill")
	}

	m.confirmSession = "alpha-1"
	m.updateConfirmKill(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after kill attempt")
	}

	m.openCloseProjectConfirm()
	if m.state != StateConfirmCloseProject || m.confirmClose == "" {
		t.Fatalf("expected confirm close project")
	}

	m.confirmClose = "Alpha"
	m.confirmCloseID = projectKey("/alpha", "Alpha")
	m.configPath = filepath.Join(t.TempDir(), "config.yml")
	m.applyCloseProject()
	if len(m.settings.HiddenProjects) == 0 {
		t.Fatalf("expected hidden projects updated")
	}
}

func TestPaneCloseAndSplitSwap(t *testing.T) {
	m := newTestModelLite()
	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}

	cmd := m.openClosePaneConfirm()
	if cmd != nil {
		t.Fatalf("expected nil cmd for running pane confirm")
	}
	if m.state != StateConfirmClosePane || m.confirmPaneSession == "" {
		t.Fatalf("expected confirm close pane")
	}
	m.updateConfirmClosePane(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.confirmPaneSession != "" || m.state != StateDashboard {
		t.Fatalf("expected reset confirm pane")
	}

	pane := m.data.Projects[0].Sessions[0].Panes[0]
	pane.Dead = true
	m.data.Projects[0].Sessions[0].Panes[0] = pane
	_ = m.openClosePaneConfirm()
	if m.state != StateDashboard || m.toast.Text == "" {
		t.Fatalf("expected dashboard state and toast for dead pane")
	}

	m.addPaneSplit(true)
	m.swapPaneWith(PaneSwapChoice{PaneIndex: "2"})
}
