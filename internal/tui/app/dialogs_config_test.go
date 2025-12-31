package app

import (
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/runenv"
)

func TestRenameDialogs(t *testing.T) {
	m := newTestModelLite()
	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}

	m.openRenameSession()
	if m.state != StateRenameSession || m.renameSession == "" {
		t.Fatalf("expected rename session state")
	}
	m.renameInput.SetValue(m.renameSession)
	m.applyRename()
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after rename session")
	}

	m.openRenamePane()
	if m.state != StateRenamePane || m.renamePaneIndex == "" {
		t.Fatalf("expected rename pane state")
	}
	m.renameInput.SetValue(m.renamePane)
	m.applyRename()
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after rename pane")
	}

	m.renameSession = ""
	m.renamePaneIndex = ""
	if _, _, ok := m.renamePaneTarget(); !ok {
		t.Fatalf("expected rename pane target fallback to selection")
	}
}

func TestProjectRootSetupAndHelp(t *testing.T) {
	t.Setenv(runenv.FreshConfigEnv, "")
	if !needsProjectRootSetup(nil, false) {
		t.Fatalf("expected project root setup needed")
	}

	m := newTestModelLite()
	m.configPath = filepath.Join(t.TempDir(), "config.yml")
	m.config = &layout.Config{}
	m.openProjectRootSetup()
	if m.state != StateProjectRootSetup {
		t.Fatalf("expected project root setup state")
	}

	tmp := t.TempDir()
	m.projectRootInput = textinput.New()
	m.projectRootInput.SetValue(tmp)
	m.applyProjectRootSetup()
	if len(m.settings.ProjectRoots) == 0 {
		t.Fatalf("expected project roots saved")
	}

	m.setState(StateHelp)
	m.updateHelp(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after help esc")
	}
}

func TestProjectRootSetupSkippedForFreshConfig(t *testing.T) {
	t.Setenv(runenv.FreshConfigEnv, "1")
	if needsProjectRootSetup(&layout.Config{}, true) {
		t.Fatalf("expected project root setup skipped when fresh config enabled")
	}
}

func TestProjectRootSetupValidationAndHelpKeys(t *testing.T) {
	m := newTestModelLite()
	m.configPath = filepath.Join(t.TempDir(), "config.yml")
	m.config = &layout.Config{}
	m.openProjectRootSetup()

	m.projectRootInput.SetValue("")
	m.applyProjectRootSetup()
	if m.toast.Text == "" {
		t.Fatalf("expected toast for empty project roots")
	}

	m.projectRootInput.SetValue(filepath.Join(t.TempDir(), "missing"))
	m.applyProjectRootSetup()
	if m.toast.Text == "" {
		t.Fatalf("expected toast for invalid project roots")
	}

	m.setState(StateHelp)
	m.updateHelp(keyRune('?'))
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after help key")
	}

	m.setState(StateHelp)
	_, cmd := m.updateHelp(keyRune('q'))
	if cmd != nil {
		t.Fatalf("expected quit prompt without cmd")
	}
	if m.state != StateConfirmQuit {
		t.Fatalf("expected confirm quit state")
	}
}

func TestHiddenProjectLabelsAndReopen(t *testing.T) {
	entry := layout.HiddenProjectConfig{Name: "Hidden", Path: "/tmp/hidden"}
	if hiddenProjectLabel(entry) == "" {
		t.Fatalf("expected hidden project label")
	}
	keys := hiddenProjectKeysFrom(entry)
	if keys.empty() || !keys.matches(entry) {
		t.Fatalf("expected hidden project key match")
	}

	m := newTestModelLite()
	m.configPath = filepath.Join(t.TempDir(), "config.yml")
	m.config = &layout.Config{
		Dashboard: layout.DashboardConfig{HiddenProjects: []layout.HiddenProjectConfig{entry}},
	}
	if len(m.hiddenProjectEntries()) == 0 {
		t.Fatalf("expected hidden project entries")
	}
	m.reopenHiddenProject(entry)
}

func TestQuickReplyFlow(t *testing.T) {
	m := newTestModelLite()
	m.selection = selectionState{ProjectID: projectKey("/alpha", "Alpha"), Session: "alpha-1", Pane: "1"}
	m.setQuickReplySize()
	_ = m.openQuickReply()
	if m.quickReplyInput.Value() != "" {
		t.Fatalf("expected quick reply input cleared")
	}

	m.quickReplyInput.SetValue("")
	m.updateQuickReply(tea.KeyMsg{Type: tea.KeyEnter})

	m.quickReplyInput.SetValue("hi")
	sendCmd := m.sendQuickReply()
	if sendCmd == nil {
		t.Fatalf("expected send cmd")
	}
	msg := sendCmd()
	if _, ok := msg.(ErrorMsg); !ok {
		t.Fatalf("expected error msg when client missing")
	}
}
