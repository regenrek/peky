package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateRenameFlows(t *testing.T) {
	m := newTestModelLite()
	m.openRenamePane()
	if m.state != StateRenamePane {
		t.Fatalf("expected rename pane state")
	}
	_, cmd := m.updateRename(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil cmd for rename with nil client")
	}
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard state for unchanged name")
	}

	m.openRenameSession()
	if m.state != StateRenameSession {
		t.Fatalf("expected rename session state")
	}
	m.updateRename(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after esc")
	}
}

func TestUpdateRenamePaneClientMissing(t *testing.T) {
	m := newTestModelLite()
	m.openRenamePane()
	if m.state != StateRenamePane {
		t.Fatalf("expected rename pane state")
	}
	m.renameInput.SetValue("new title")
	_, cmd := m.updateRename(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil cmd when client missing")
	}
	if m.toast.Text == "" {
		t.Fatalf("expected toast for rename failure")
	}
	if m.state != StateRenamePane {
		t.Fatalf("expected rename pane state to remain on error")
	}
}
