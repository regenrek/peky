package peakypanes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/layout"
)

func TestModelInitAndRefreshHelpers(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.beginRefresh()
	if !m.refreshing || m.refreshInFlight == 0 {
		t.Fatalf("beginRefresh() did not set flags")
	}
	m.endRefresh()
	if m.refreshing {
		t.Fatalf("endRefresh() did not clear flags")
	}
	if cmd := tickCmd(10 * time.Millisecond); cmd == nil {
		t.Fatalf("tickCmd() returned nil")
	}
	if cmd := m.selectionRefreshCmd(); cmd == nil {
		t.Fatalf("selectionRefreshCmd() returned nil")
	}
	if cmd := m.Init(); cmd == nil {
		t.Fatalf("Init() returned nil")
	}
}

func TestUpdateHandlesMessages(t *testing.T) {
	m, _ := newTestModel(t, nil)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
	_, _ = m.Update(refreshTickMsg{})
	_, _ = m.Update(selectionRefreshMsg{Version: m.selectionVersion})

	result := tmuxSnapshotResult{Data: DashboardData{Projects: []ProjectGroup{{Name: "Proj"}}}, Settings: m.settings, Version: m.selectionVersion}
	_, _ = m.Update(tmuxSnapshotMsg{Result: result})
	_, _ = m.Update(SuccessMsg{Message: "ok"})
	_, _ = m.Update(WarningMsg{Message: "warn"})
	_, _ = m.Update(InfoMsg{Message: "info"})
	_, _ = m.Update(ErrorMsg{Err: errTest("boom")})
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestUpdateDashboardKeys(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:         "sess",
			Status:       StatusRunning,
			ActiveWindow: "1",
			Path:         m.configPath,
			Windows: []WindowItem{{
				Index: "1",
				Panes: []PaneItem{{Index: "0", Active: true}},
			}},
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess", Window: "1"}

	m.filterActive = true
	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyEsc})
	m.filterActive = false

	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	_, _ = m.updateDashboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
}

func TestPickersAndRenamePaths(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:         "sess",
			Status:       StatusRunning,
			ActiveWindow: "1",
			Path:         m.configPath,
			Windows: []WindowItem{{
				Index: "1",
				Name:  "win",
				Panes: []PaneItem{{Index: "0", Title: "pane"}},
			}},
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess", Window: "1"}

	m.openRenameSession()
	m.openRenameWindow()
	m.openRenamePane()
	_, _ = m.updateRename(tea.KeyMsg{Type: tea.KeyEsc})

	m.openProjectPicker()
	m.projectPicker.SetItems([]list.Item{GitProject{Name: "repo", Path: "/tmp"}})
	_, _ = m.updateProjectPicker(tea.KeyMsg{Type: tea.KeyEnter})
	m.state = StateProjectPicker
	_, _ = m.updateProjectPicker(tea.KeyMsg{Type: tea.KeyEsc})

	m.layoutPicker.SetItems([]list.Item{LayoutChoice{Label: "dev", LayoutName: "dev"}})
	m.state = StateLayoutPicker
	_, _ = m.updateLayoutPicker(tea.KeyMsg{Type: tea.KeyEsc})

	m.commandPalette.SetItems(m.commandPaletteItems())
	_, _ = m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEsc})
}

func TestOpenProjectSelectsProjectAndSession(t *testing.T) {
	m, _ := newTestModel(t, nil)
	projectPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectPath, ".peakypanes.yml"), []byte("session: My Session\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	m.config = &layout.Config{Projects: []layout.ProjectConfig{{
		Name: "Configured",
		Path: projectPath,
	}}}

	m.openProjectPicker()
	m.projectPicker.SetItems([]list.Item{GitProject{Name: "nested/repo", Path: projectPath}})
	_, _ = m.updateProjectPicker(tea.KeyMsg{Type: tea.KeyEnter})

	if m.selection.Project != "Configured" {
		t.Fatalf("selection.Project = %q, want %q", m.selection.Project, "Configured")
	}
	if m.selection.Session != "My Session" {
		t.Fatalf("selection.Session = %q, want %q", m.selection.Session, "My Session")
	}
}

func TestMiscHelpers(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.setLayoutPickerSize()
	m.setCommandPaletteSize()
	m.setQuickReplySize()
	m.toggleWindows()

	if out := layoutChoicesToItems([]LayoutChoice{{Label: "dev"}}); len(out) != 1 {
		t.Fatalf("layoutChoicesToItems() = %#v", out)
	}
	if summary := layoutSummary(&layout.LayoutConfig{Grid: "2x2"}); summary == "" {
		t.Fatalf("layoutSummary() empty")
	}
	if tmuxSocketFromEnv("/tmp/tmux,123") != "/tmp/tmux" {
		t.Fatalf("tmuxSocketFromEnv() failed")
	}

	cmd := m.openNewTerminal("echo", []string{"hi"}, "ok")
	if cmd == nil {
		t.Fatalf("openNewTerminal() returned nil")
	}
	if focus := m.focusTerminalApp("msg"); focus == nil {
		t.Fatalf("focusTerminalApp() returned nil")
	}
	if m.newTerminalCommand("echo", nil) == nil {
		t.Fatalf("newTerminalCommand() returned nil")
	}
	_ = m.focusTerminalCommand()

	if view := m.viewDashboard(); strings.TrimSpace(view) == "" {
		t.Fatalf("viewDashboard() empty")
	}
}

func TestStartNewSessionWithLayout(t *testing.T) {
	specs := []cmdSpec{{name: "tmux", args: []string{"list-sessions", "-F", "#{session_name}"}, exit: 0}}
	m, runner := newTestModel(t, specs)
	root := t.TempDir()
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Path: root,
		Sessions: []SessionItem{{
			Name:   "sess",
			Status: StatusRunning,
			Path:   root,
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess"}
	cmd := m.startNewSessionWithLayout("dev-3")
	if cmd == nil {
		t.Fatalf("startNewSessionWithLayout() returned nil")
	}
	runner.assertDone()
}
