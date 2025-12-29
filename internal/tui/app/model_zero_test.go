package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/tui/picker"
)

func TestUpdateHelpAndProjectRootSetup(t *testing.T) {
	m := newTestModel(t)
	m.state = StateHelp
	m.updateHelp(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("updateHelp() did not return to dashboard")
	}

	m.state = StateProjectRootSetup
	m.updateProjectRootSetup(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("updateProjectRootSetup() did not return to dashboard")
	}
}

func TestOpenLayoutAndCommandPalette(t *testing.T) {
	m := newTestModel(t)
	root := t.TempDir()
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:       projectKey(root, "Proj"),
		Name:     "Proj",
		Path:     root,
		Sessions: []SessionItem{{Name: "sess", Status: StatusRunning, Path: root}},
	}}}
	m.selection = selectionState{ProjectID: projectKey(root, "Proj"), Session: "sess"}

	m.openLayoutPicker()
	if m.state != StateLayoutPicker {
		t.Fatalf("openLayoutPicker() state = %v", m.state)
	}

	cmd := m.openCommandPalette()
	if cmd == nil {
		t.Fatalf("openCommandPalette() returned nil")
	}
	if m.state != StateCommandPalette {
		t.Fatalf("openCommandPalette() state = %v", m.state)
	}
}

func TestAttachOrStart(t *testing.T) {
	m := newTestModel(t)
	root := t.TempDir()
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:   projectKey(root, "Proj"),
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:   "sess",
			Status: StatusStopped,
			Path:   root,
		}},
	}}}
	m.selection = selectionState{ProjectID: projectKey(root, "Proj"), Session: "sess"}
	if cmd := m.attachOrStart(); cmd == nil {
		t.Fatalf("attachOrStart() returned nil for stopped session")
	}

	m.data.Projects[0].Sessions[0].Status = StatusRunning
	m.terminalFocus = false
	if cmd := m.attachOrStart(); cmd != nil {
		t.Fatalf("attachOrStart() for running session should not return cmd")
	}
	if !m.terminalFocus {
		t.Fatalf("attachOrStart() should enable terminal focus")
	}
}

func TestEditConfigAndCommandPaletteEnter(t *testing.T) {
	m := newTestModel(t)
	if cmd := m.editConfig(); cmd == nil {
		t.Fatalf("editConfig() returned nil")
	}

	m.commandPalette.SetItems([]list.Item{picker.CommandItem{Label: "Run", Desc: "do", Run: func() tea.Cmd { return NewInfoCmd("ok") }}})
	m.state = StateCommandPalette
	_, cmd := m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("updateCommandPalette(enter) returned nil")
	}
}
