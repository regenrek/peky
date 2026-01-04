package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/tui/picker"
	"github.com/regenrek/peakypanes/internal/tui/views"
)

func TestModelInitAndRefreshHelpers(t *testing.T) {
	m := newTestModel(t)
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
	m := newTestModel(t)
	_, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
	_, _ = m.Update(refreshTickMsg{})
	_, _ = m.Update(selectionRefreshMsg{Version: m.selectionVersion})

	result := dashboardSnapshotResult{Data: DashboardData{Projects: []ProjectGroup{{ID: projectKey("", "Proj"), Name: "Proj"}}}, Settings: m.settings, Version: m.selectionVersion}
	_, _ = m.Update(dashboardSnapshotMsg{Result: result})
	_, _ = m.Update(SuccessMsg{Message: "ok"})
	_, _ = m.Update(WarningMsg{Message: "warn"})
	_, _ = m.Update(InfoMsg{Message: "info"})
	_, _ = m.Update(ErrorMsg{Err: errTest("boom")})
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestUpdateDashboardKeys(t *testing.T) {
	m := newTestModel(t)
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:   projectKey("", "Proj"),
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:       "sess",
			Status:     StatusRunning,
			ActivePane: "0",
			Path:       m.configPath,
			Panes:      []PaneItem{{Index: "0", Active: true}},
		}},
	}}}
	m.selection = selectionState{ProjectID: projectKey("", "Proj"), Session: "sess", Pane: "0"}

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
	m := newTestModel(t)
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:   projectKey("", "Proj"),
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:       "sess",
			Status:     StatusRunning,
			ActivePane: "0",
			Path:       m.configPath,
			Panes:      []PaneItem{{Index: "0", Title: "pane"}},
		}},
	}}}
	m.selection = selectionState{ProjectID: projectKey("", "Proj"), Session: "sess", Pane: "0"}

	m.openRenameSession()
	m.openRenamePane()
	_, _ = m.updateRename(tea.KeyMsg{Type: tea.KeyEsc})

	m.openProjectPicker()
	m.projectPicker.SetItems([]list.Item{picker.ProjectItem{Name: "repo", Path: "/tmp"}})
	_, _ = m.updateProjectPicker(tea.KeyMsg{Type: tea.KeyEnter})
	m.state = StateProjectPicker
	_, _ = m.updateProjectPicker(tea.KeyMsg{Type: tea.KeyEsc})

	m.layoutPicker.SetItems([]list.Item{picker.LayoutChoice{Label: "dev", LayoutName: "dev"}})
	m.state = StateLayoutPicker
	_, _ = m.updateLayoutPicker(tea.KeyMsg{Type: tea.KeyEsc})

	m.commandPalette.SetItems(m.commandPaletteItems())
	_, _ = m.updateCommandPalette(tea.KeyMsg{Type: tea.KeyEsc})
}

func TestOpenProjectSelectsProjectAndSession(t *testing.T) {
	m := newTestModel(t)
	projectPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectPath, ".peakypanes.yml"), []byte("session: My Session\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	m.config = &layout.Config{Projects: []layout.ProjectConfig{{
		Name: "Configured",
		Path: projectPath,
	}}}

	m.openProjectPicker()
	m.projectPicker.SetItems([]list.Item{picker.ProjectItem{Name: "nested/repo", Path: projectPath}})
	_, _ = m.updateProjectPicker(tea.KeyMsg{Type: tea.KeyEnter})

	if m.selection.ProjectID != projectKey(projectPath, "Configured") {
		t.Fatalf("selection.ProjectID = %q, want %q", m.selection.ProjectID, projectKey(projectPath, "Configured"))
	}
	if m.selection.Session != "My Session" {
		t.Fatalf("selection.Session = %q, want %q", m.selection.Session, "My Session")
	}
}

func TestMiscHelpers(t *testing.T) {
	m := newTestModel(t)
	m.setLayoutPickerSize()
	m.setCommandPaletteSize()
	m.setQuickReplySize()
	m.togglePanes()

	if out := layoutChoicesToItems([]picker.LayoutChoice{{Label: "dev"}}); len(out) != 1 {
		t.Fatalf("layoutChoicesToItems() = %#v", out)
	}
	if summary := layoutSummary(&layout.LayoutConfig{Grid: "2x2"}); summary == "" {
		t.Fatalf("layoutSummary() empty")
	}

	if view := views.Render(m.viewModel()); strings.TrimSpace(view) == "" {
		t.Fatalf("views.Render() empty")
	}
}

func TestStartNewSessionWithLayout(t *testing.T) {
	m := newTestModel(t)
	root := t.TempDir()
	m.data = DashboardData{Projects: []ProjectGroup{{
		ID:   projectKey(root, "Proj"),
		Name: "Proj",
		Path: root,
		Sessions: []SessionItem{{
			Name:   "sess",
			Status: StatusRunning,
			Path:   root,
		}},
	}}}
	m.selection = selectionState{ProjectID: projectKey(root, "Proj"), Session: "sess"}
	cmd := m.startNewSessionWithLayout(layout.DefaultLayoutName)
	if cmd == nil {
		t.Fatalf("startNewSessionWithLayout() returned nil")
	}
}
