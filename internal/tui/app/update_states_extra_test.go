package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestHandleKeyMsgAdditionalStates(t *testing.T) {
	m := newTestModelLite()

	m.state = StateConfirmCloseProject
	m.confirmClose = "Alpha"
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}); !handled {
		t.Fatalf("expected close project key handled")
	}
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after cancel")
	}

	m.state = StateConfirmCloseAllProjects
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}); !handled {
		t.Fatalf("expected close all projects key handled")
	}
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after cancel")
	}

	m.state = StateConfirmClosePane
	m.confirmPaneSession = "alpha-1"
	m.confirmPaneIndex = "1"
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}); !handled {
		t.Fatalf("expected close pane key handled")
	}

	m.openRenameSession()
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected rename key handled")
	}

	m.config = &layout.Config{}
	m.openProjectRootSetup()
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected project root setup handled")
	}

	m.state = StateLayoutPicker
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected layout picker handled")
	}

	m.state = StatePaneSplitPicker
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected pane split picker handled")
	}

	m.state = StatePaneSwapPicker
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected pane swap picker handled")
	}

	m.state = StateCommandPalette
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected command palette handled")
	}

	m.state = StateSettingsMenu
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected settings menu handled")
	}

	m.state = StateDebugMenu
	if _, _, handled := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc}); !handled {
		t.Fatalf("expected debug menu handled")
	}
}
