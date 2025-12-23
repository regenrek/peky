package peakypanes

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateHelpAndProjectRootSetup(t *testing.T) {
	m, _ := newTestModel(t, nil)
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
	m, _ := newTestModel(t, nil)
	root := t.TempDir()
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name:     "Proj",
		Path:     root,
		Sessions: []SessionItem{{Name: "sess", Status: StatusRunning, Path: root}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess"}

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

func TestAttachOrStartAndSessionActions(t *testing.T) {
	specs := []cmdSpec{{name: "tmux", args: []string{"list-clients", "-t", "sess", "-F", "#{client_tty}"}, stdout: "", exit: 0}}
	m, runner := newTestModel(t, specs)
	root := t.TempDir()
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:   "sess",
			Status: StatusStopped,
			Path:   root,
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess"}
	if cmd := m.attachOrStart(); cmd == nil {
		t.Fatalf("attachOrStart() returned nil")
	}

	m.insideTmux = true
	if cmd := m.attachSession(SessionItem{Name: "sess"}); cmd == nil {
		t.Fatalf("attachSession() returned nil")
	}

	if cmd := m.startProject(SessionItem{Name: "sess", Path: root, LayoutName: "dev"}); cmd == nil {
		t.Fatalf("startProject() returned nil")
	}

	m.data.Projects[0].Sessions[0].Status = StatusRunning
	if cmd := m.openSessionInNewTerminal(false); cmd == nil {
		t.Fatalf("openSessionInNewTerminal() returned nil")
	}
	runner.assertDone()
}

func TestEditConfigAndCommandPaletteEnter(t *testing.T) {
	m, _ := newTestModel(t, nil)
	if cmd := m.editConfig(); cmd == nil {
		t.Fatalf("editConfig() returned nil")
	}

	m.commandPalette.SetItems([]list.Item{CommandItem{Label: "Run", Desc: "do", Run: func(*Model) tea.Cmd { return NewInfoCmd("ok") }}})
	m.state = StateCommandPalette
	_, cmd := m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("updateCommandPalette(enter) returned nil")
	}
}

func TestTypesHelpers(t *testing.T) {
	gp := GitProject{Name: "repo", Path: "/tmp/repo"}
	if gp.Description() == "" || gp.FilterValue() == "" {
		t.Fatalf("GitProject helpers empty")
	}
	lc := LayoutChoice{Label: "dev", Desc: "layout"}
	if lc.Description() == "" || lc.FilterValue() == "" {
		t.Fatalf("LayoutChoice helpers empty")
	}
	ci := CommandItem{Label: "Cmd", Desc: "desc"}
	if !strings.Contains(ci.FilterValue(), "cmd") {
		t.Fatalf("CommandItem.FilterValue() = %q", ci.FilterValue())
	}
}
