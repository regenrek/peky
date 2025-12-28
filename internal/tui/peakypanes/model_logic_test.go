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

func TestSendQuickReplyNative(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "sess")
	if len(snap.Panes) == 0 {
		t.Fatalf("session snapshot missing panes")
	}
	paneSnap := snap.Panes[0]

	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:       snap.Name,
			Status:     StatusRunning,
			ActivePane: paneSnap.Index,
			Panes: []PaneItem{{
				ID:     paneSnap.ID,
				Index:  paneSnap.Index,
				Title:  paneSnap.Title,
				Active: true,
			}},
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: snap.Name, Pane: paneSnap.Index}

	m.quickReplyInput.SetValue("hello")
	cmd := m.sendQuickReply()
	msg := cmd()
	if _, ok := msg.(SuccessMsg); !ok {
		t.Fatalf("sendQuickReply() msg = %#v", msg)
	}
}

func TestSendQuickReplyEmpty(t *testing.T) {
	m := newTestModel(t)
	m.quickReplyInput.SetValue(" ")
	msg := m.sendQuickReply()()
	if _, ok := msg.(InfoMsg); !ok {
		t.Fatalf("sendQuickReply() msg = %#v", msg)
	}
}

func TestRenameSession(t *testing.T) {
	m := newTestModel(t)
	startNativeSession(t, m, "old")

	m.selection = selectionState{Project: "Proj", Session: "old"}
	m.expandedSessions["old"] = true
	m.renameSession = "old"
	m.state = StateRenameSession
	m.renameInput = textinput.New()
	m.renameInput.SetValue("new")
	m.applyRename()

	if m.selection.Session != "new" {
		t.Fatalf("selection.Session = %q", m.selection.Session)
	}
	if !sessionExists(t, m.client, "new") {
		t.Fatalf("expected renamed session")
	}
}

func TestRenamePane(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "sess")
	if len(snap.Panes) == 0 {
		t.Fatalf("session snapshot missing panes")
	}
	paneSnap := snap.Panes[0]

	m.renameSession = "sess"
	m.renamePaneIndex = paneSnap.Index
	m.renamePane = paneSnap.Title
	m.state = StateRenamePane
	m.renameInput = textinput.New()
	m.renameInput.SetValue("pane")
	m.applyRename()

	sessionSnap := waitForSessionSnapshot(t, m.client, "sess")
	if len(sessionSnap.Panes) == 0 {
		t.Fatalf("pane rename failed: %#v", sessionSnap)
	}
	if sessionSnap.Panes[0].Title != "pane" {
		t.Fatalf("pane title = %q", sessionSnap.Panes[0].Title)
	}
}

func TestUpdateConfirmKill(t *testing.T) {
	m := newTestModel(t)
	startNativeSession(t, m, "sess")
	m.confirmSession = "sess"
	m.confirmProject = "Proj"
	m.state = StateConfirmKill

	_, _ = m.updateConfirmKill(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.state != StateDashboard {
		t.Fatalf("state = %v", m.state)
	}
	if sessionExists(t, m.client, "sess") {
		t.Fatalf("session should be killed")
	}
}

func TestUpdateConfirmCloseProject(t *testing.T) {
	m := newTestModel(t)
	projectPath := t.TempDir()
	cfg := &layout.Config{Projects: []layout.ProjectConfig{{
		Name: "Proj",
		Path: projectPath,
	}}}
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := layout.SaveConfig(m.configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	m.config = cfg
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name:       "Proj",
		Path:       projectPath,
		FromConfig: true,
		Sessions:   []SessionItem{{Name: "sess", Status: StatusRunning}},
	}}}
	m.confirmClose = "Proj"
	m.state = StateConfirmCloseProject

	_, _ = m.updateConfirmCloseProject(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.state != StateDashboard {
		t.Fatalf("state = %v", m.state)
	}
	loaded, err := layout.LoadConfig(m.configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(loaded.Projects) != 1 {
		t.Fatalf("Projects = %#v", loaded.Projects)
	}
	if len(loaded.Dashboard.HiddenProjects) != 1 {
		t.Fatalf("HiddenProjects = %#v", loaded.Dashboard.HiddenProjects)
	}
	entry := loaded.Dashboard.HiddenProjects[0]
	if normalizeProjectKey(entry.Path, entry.Name) != projectKey(projectPath, "Proj") {
		t.Fatalf("HiddenProjects[0] = %#v", entry)
	}
}

func TestUpdateConfirmCloseProjectKillsSessions(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "sess")
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Sessions: []SessionItem{{
			Name:   snap.Name,
			Status: StatusRunning,
		}},
	}}}
	m.confirmClose = "Proj"
	m.state = StateConfirmCloseProject

	_, _ = m.updateConfirmCloseProject(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.state != StateDashboard {
		t.Fatalf("state = %v", m.state)
	}
	if sessionExists(t, m.client, snap.Name) {
		t.Fatalf("session should be killed")
	}
}

func TestAddPaneSplit(t *testing.T) {
	m := newTestModel(t)
	snap := startNativeSession(t, m, "sess")
	if len(snap.Panes) == 0 {
		t.Fatalf("session snapshot missing panes")
	}
	paneSnap := snap.Panes[0]
	m.data = DashboardData{Projects: []ProjectGroup{{
		Name: "Proj",
		Path: snap.Path,
		Sessions: []SessionItem{{
			Name:       snap.Name,
			Status:     StatusRunning,
			Path:       snap.Path,
			ActivePane: paneSnap.Index,
			Panes: []PaneItem{{
				ID:     paneSnap.ID,
				Index:  paneSnap.Index,
				Title:  paneSnap.Title,
				Active: true,
			}},
		}},
	}}}
	m.selection = selectionState{Project: "Proj", Session: snap.Name, Pane: paneSnap.Index}

	_ = m.addPaneSplit(false)

	sessionSnap := waitForSessionSnapshot(t, m.client, snap.Name)
	if len(sessionSnap.Panes) < 2 {
		t.Fatalf("expected new pane, got %#v", sessionSnap)
	}
}

func TestProjectRootsSetup(t *testing.T) {
	m := newTestModel(t)
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
	m := newTestModel(t)
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
	m := newTestModel(t)
	choices, err := m.loadLayoutChoices("")
	if err != nil {
		t.Fatalf("loadLayoutChoices() error: %v", err)
	}
	if len(choices) == 0 || !strings.Contains(strings.ToLower(choices[0].Label), "auto") {
		t.Fatalf("expected auto choice, got %#v", choices)
	}
}
