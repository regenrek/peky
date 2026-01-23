package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestHandleKeyMsgAdditionalStates(t *testing.T) {
	t.Run("close and quit cancels", func(t *testing.T) {
		m := newTestModelLite()

		m.state = StateConfirmCloseProject
		m.confirmClose = "Alpha"
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		if m.state != StateDashboard {
			t.Fatalf("expected dashboard after cancel")
		}

		m.state = StateConfirmCloseAllProjects
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		if m.state != StateDashboard {
			t.Fatalf("expected dashboard after cancel")
		}

		m.state = StateConfirmQuit
		m.confirmQuitRunning = 1
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		if m.state != StateDashboard {
			t.Fatalf("expected dashboard after quit cancel")
		}
	})

	t.Run("close pane cancel", func(t *testing.T) {
		m := newTestModelLite()
		m.state = StateConfirmClosePane
		m.confirmPaneSession = "alpha-1"
		m.confirmPaneIndex = "1"
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	})

	t.Run("esc closes dialogs", func(t *testing.T) {
		m := newTestModelLite()

		m.openRenameSession()
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.config = &layout.Config{}
		m.openProjectRootSetup()
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.state = StateLayoutPicker
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.state = StatePaneSplitPicker
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.state = StatePaneSwapPicker
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.state = StateCommandPalette
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.state = StateSettingsMenu
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.state = StatePerformanceMenu
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.state = StateDebugMenu
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})

		m.state = StatePaneColor
		assertHandledKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	})
}

func assertHandledKey(t *testing.T, m *Model, msg tea.KeyMsg) {
	t.Helper()
	if _, _, handled := m.handleKeyMsg(msg); !handled {
		t.Fatalf("expected key handled")
	}
}
