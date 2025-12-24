package peakypanes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/layout"
)

func TestSelectedPaneTargetAndQuickReply(t *testing.T) {
	specs := []cmdSpec{
		{
			name: "tmux",
			args: []string{"send-keys", "-t", "sess:1.1", "-l", "hello"},
			exit: 0,
		},
		{
			name: "tmux",
			args: []string{"send-keys", "-t", "sess:1.1", "Enter"},
			exit: 0,
		},
	}
	m, runner := newTestModel(t, specs)

	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:         "sess",
			Status:       StatusRunning,
			ActiveWindow: "1",
			Windows: []WindowItem{{
				Index:  "1",
				Name:   "main",
				Active: true,
				Panes: []PaneItem{
					{Index: "0", Title: "shell", Active: true},
					{Index: "1", Title: "worker", Active: false},
				},
			}},
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess", Window: "1", Pane: "1"}

	if target, label, ok := m.selectedPaneTarget(); !ok || target != "sess:1.1" || label != "worker" {
		t.Fatalf("selectedPaneTarget() = %q,%q,%v", target, label, ok)
	}

	m.quickReplyInput.SetValue("hello")
	cmd := m.sendQuickReply()
	msg := cmd()
	if _, ok := msg.(SuccessMsg); !ok {
		t.Fatalf("sendQuickReply() msg = %#v", msg)
	}

	runner.assertDone()
}

func TestSendQuickReplyEmpty(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.quickReplyInput.SetValue(" ")
	msg := m.sendQuickReply()()
	if _, ok := msg.(InfoMsg); !ok {
		t.Fatalf("sendQuickReply() msg = %#v", msg)
	}
}

func TestRenameSessionWindowAndPane(t *testing.T) {
	specs := []cmdSpec{
		{name: "tmux", args: []string{"rename-session", "-t", "old", "new"}, exit: 0},
		{name: "tmux", args: []string{"rename-window", "-t", "sess:1", "win"}, exit: 0},
		{name: "tmux", args: []string{"select-pane", "-t", "sess:1.0", "-T", "pane"}, exit: 0},
	}
	m, runner := newTestModel(t, specs)
	m.selection = selectionState{Project: "Proj", Session: "old", Window: "1"}
	m.expandedSessions["old"] = true

	m.renameSession = "old"
	m.state = StateRenameSession
	m.renameInput = textinput.New()
	m.renameInput.SetValue("new")
	m.applyRename()
	if m.selection.Session != "new" {
		t.Fatalf("selection.Session = %q", m.selection.Session)
	}

	m.renameSession = "sess"
	m.renameWindowIndex = "1"
	m.renameWindow = "oldwin"
	m.state = StateRenameWindow
	m.renameInput = textinput.New()
	m.renameInput.SetValue("win")
	m.applyRename()

	m.renameSession = "sess"
	m.renameWindowIndex = "1"
	m.renamePane = "oldpane"
	m.renamePaneIndex = "0"
	m.state = StateRenamePane
	m.renameInput = textinput.New()
	m.renameInput.SetValue("pane")
	m.applyRename()

	runner.assertDone()
}

func TestUpdateConfirmKill(t *testing.T) {
	specs := []cmdSpec{{name: "tmux", args: []string{"kill-session", "-t", "sess"}, exit: 0}}
	m, runner := newTestModel(t, specs)
	m.confirmSession = "sess"
	m.confirmProject = "Proj"
	m.state = StateConfirmKill

	_, _ = m.updateConfirmKill(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.state != StateDashboard {
		t.Fatalf("state = %v", m.state)
	}
	runner.assertDone()
}

func TestUpdateConfirmCloseProject(t *testing.T) {
	specs := []cmdSpec{{name: "tmux", args: []string{"kill-session", "-t", "sess"}, exit: 0}}
	m, runner := newTestModel(t, specs)
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name:     "Proj",
		Sessions: []SessionItem{{Name: "sess", Status: StatusRunning}},
	}}}
	m.confirmClose = "Proj"
	m.state = StateConfirmCloseProject

	_, _ = m.updateConfirmCloseProject(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.state != StateDashboard {
		t.Fatalf("state = %v", m.state)
	}
	runner.assertDone()
}

func TestOpenNewWindow(t *testing.T) {
	tmpDir := t.TempDir()
	specs := []cmdSpec{{name: "tmux", args: []string{"new-window", "-t", "sess", "-c", tmpDir}, exit: 0}}
	m, runner := newTestModel(t, specs)
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Path: tmpDir,
		Sessions: []SessionItem{{
			Name:         "sess",
			Status:       StatusRunning,
			Path:         tmpDir,
			ActiveWindow: "1",
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess", Window: "1"}

	_ = m.openNewWindow()

	runner.assertDone()
}

func TestProjectRootsSetup(t *testing.T) {
	m, _ := newTestModel(t, nil)
	root := t.TempDir()
	m.configPath = filepath.Join(t.TempDir(), "config.yml")
	m.projectRootInput = textinput.New()
	m.projectRootInput.SetValue(root)
	m.applyProjectRootSetup()
	if len(m.settings.ProjectRoots) != 1 || m.settings.ProjectRoots[0] != root {
		t.Fatalf("ProjectRoots = %#v", m.settings.ProjectRoots)
	}
}

func TestScanGitProjects(t *testing.T) {
	m, _ := newTestModel(t, nil)
	root := t.TempDir()
	project := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	m.settings.ProjectRoots = []string{root}
	m.scanGitProjects()
	if len(m.gitProjects) != 1 {
		t.Fatalf("gitProjects = %#v", m.gitProjects)
	}
	items := m.gitProjectsToItems()
	if len(items) != 1 {
		t.Fatalf("gitProjectsToItems() = %#v", items)
	}
}

func TestLoadLayoutChoicesIncludesAuto(t *testing.T) {
	m, _ := newTestModel(t, nil)
	choices, err := m.loadLayoutChoices("")
	if err != nil {
		t.Fatalf("loadLayoutChoices() error: %v", err)
	}
	if len(choices) == 0 || !strings.Contains(choices[0].Label, "auto") {
		t.Fatalf("loadLayoutChoices() = %#v", choices)
	}
}

func TestFilteredSessions(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.filterInput.SetValue("api")
	list := []SessionItem{
		{Name: "api", Path: "/srv/api"},
		{Name: "web", Path: "/srv/web"},
	}
	out := m.filteredSessions(list)
	if len(out) != 1 || out[0].Name != "api" {
		t.Fatalf("filteredSessions() = %#v", out)
	}
}

func TestNeedsProjectRootSetup(t *testing.T) {
	cfg := &layout.Config{}
	if !needsProjectRootSetup(cfg, false) {
		t.Fatalf("needsProjectRootSetup() expected true")
	}
	cfg.Dashboard.ProjectRoots = []string{"/tmp"}
	if needsProjectRootSetup(cfg, true) {
		t.Fatalf("needsProjectRootSetup() expected false")
	}
}

func TestSelectHelpers(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.tab = TabProject
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name:     "A",
		Sessions: []SessionItem{{Name: "s1", ActiveWindow: "0", Windows: []WindowItem{{Index: "0", Panes: []PaneItem{{Index: "0", Active: true}, {Index: "1"}}}}}},
	}, {
		Name:     "B",
		Sessions: []SessionItem{{Name: "s2", ActiveWindow: "1", Windows: []WindowItem{{Index: "1", Panes: []PaneItem{{Index: "0", Active: true}}}}}},
	}}}
	m.selection = selectionState{Project: "A", Session: "s1", Window: "0"}
	m.selectTab(1)
	if m.selection.Project != "B" {
		t.Fatalf("selectTab() = %q", m.selection.Project)
	}
	m.selectSession(1)
	if m.selection.Session == "" {
		t.Fatalf("selectSession() empty")
	}
	m.selectWindow(1)
	m.selectPane(1)
}

func TestSelectionMemoryAcrossProjects(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.tab = TabProject
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "A",
		Sessions: []SessionItem{
			{Name: "s1", ActiveWindow: "0", Windows: []WindowItem{{Index: "0", Panes: []PaneItem{{Index: "0", Active: true}}}}},
			{Name: "s2", ActiveWindow: "1", Windows: []WindowItem{{Index: "1", Panes: []PaneItem{{Index: "0", Active: true}}}}},
		},
	}, {
		Name: "B",
		Sessions: []SessionItem{
			{Name: "b1", ActiveWindow: "0", Windows: []WindowItem{{Index: "0", Panes: []PaneItem{{Index: "0", Active: true}}}}},
			{Name: "b2", ActiveWindow: "1", Windows: []WindowItem{{Index: "1", Panes: []PaneItem{{Index: "0", Active: true}}}}},
		},
	}}}
	m.selection = selectionState{Project: "A", Session: "s1", Window: "0"}
	m.selectSession(1)
	if m.selection.Session != "s2" {
		t.Fatalf("selectSession() = %q", m.selection.Session)
	}

	m.selectTab(1)
	if m.selection.Project != "B" {
		t.Fatalf("selectTab() = %q", m.selection.Project)
	}
	m.selectSession(1)
	if m.selection.Session != "b2" {
		t.Fatalf("selectSession() = %q", m.selection.Session)
	}

	m.selectTab(-1)
	if m.selection.Project != "A" || m.selection.Session != "s2" {
		t.Fatalf("selection restore = %#v", m.selection)
	}
}

func TestSelectionMemoryFallbackWhenMissing(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.tab = TabProject
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "A",
		Sessions: []SessionItem{
			{Name: "s1", ActiveWindow: "0", Windows: []WindowItem{{Index: "0", Panes: []PaneItem{{Index: "0", Active: true}}}}},
			{Name: "s2", ActiveWindow: "1", Windows: []WindowItem{{Index: "1", Panes: []PaneItem{{Index: "0", Active: true}}}}},
		},
	}, {
		Name: "B",
		Sessions: []SessionItem{
			{Name: "b1", ActiveWindow: "0", Windows: []WindowItem{{Index: "0", Panes: []PaneItem{{Index: "0", Active: true}}}}},
		},
	}}}
	m.selection = selectionState{Project: "A", Session: "s2", Window: "1"}
	m.rememberSelection(m.selection)

	m.selectTab(1)
	if m.selection.Project != "B" {
		t.Fatalf("selectTab() = %q", m.selection.Project)
	}

	m.data = DashboardData{Projects: []ProjectGroup{{
		Name:     "A",
		Sessions: []SessionItem{{Name: "s1", ActiveWindow: "0", Windows: []WindowItem{{Index: "0", Panes: []PaneItem{{Index: "0", Active: true}}}}}},
	}, {
		Name:     "B",
		Sessions: []SessionItem{{Name: "b1", ActiveWindow: "0", Windows: []WindowItem{{Index: "0", Panes: []PaneItem{{Index: "0", Active: true}}}}}},
	}}}

	m.selectTab(-1)
	if m.selection.Project != "A" || m.selection.Session != "s1" {
		t.Fatalf("selection fallback = %#v", m.selection)
	}
}

func TestOpenQuickReply(t *testing.T) {
	m, _ := newTestModel(t, nil)
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:         "sess",
			Status:       StatusRunning,
			ActiveWindow: "1",
			Windows: []WindowItem{{
				Index: "1",
				Panes: []PaneItem{{Index: "0", Active: true}},
			}},
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: "sess", Window: "1"}
	m.openQuickReply()
	if !m.quickReplyInput.Focused() {
		t.Fatalf("openQuickReply() did not focus input")
	}
	_, _ = m.updateQuickReply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_, cmd := m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("updateQuickReply() expected cmd")
	}
}

func TestParseAndValidateProjectRoots(t *testing.T) {
	root := t.TempDir()
	roots := parseProjectRoots(root + ", " + "/missing")
	if len(roots) != 2 {
		t.Fatalf("parseProjectRoots() = %#v", roots)
	}
	valid, invalid := validateProjectRoots(roots)
	if len(valid) != 1 || len(invalid) != 1 {
		t.Fatalf("validateProjectRoots() valid=%#v invalid=%#v", valid, invalid)
	}
}
