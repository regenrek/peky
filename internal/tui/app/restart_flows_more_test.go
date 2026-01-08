package app

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestUpdateRestartNoticeStartFreshClearsFlag(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := layout.SaveConfig(cfgPath, &layout.Config{}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	m := newTestModelLite()
	m.configPath = cfgPath
	m.restartNoticePending = true
	m.setState(StateRestartNotice)

	_, _ = m.updateRestartNotice(tea.KeyMsg{Type: tea.KeyEnter})
	if m.restartNoticePending {
		t.Fatalf("expected restartNoticePending=false")
	}
	if m.state != StateLayoutPicker && m.state != StateProjectPicker && m.state != StateDashboard {
		t.Fatalf("unexpected state=%v", m.state)
	}
}

func TestUpdateRestartNoticeCheckStaleShowsToast(t *testing.T) {
	m := newTestModelLite()
	m.restartNoticePending = true
	m.setState(StateRestartNotice)

	_, _ = m.updateRestartNotice(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.state != StateDashboard {
		t.Fatalf("state=%v want=%v", m.state, StateDashboard)
	}
	if m.toast.Text == "" {
		t.Fatalf("expected toast")
	}
}

func TestUpdateConfirmRestartCancel(t *testing.T) {
	m := newTestModelLite()
	m.setState(StateConfirmRestart)
	_, _ = m.updateConfirmRestart(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateDashboard {
		t.Fatalf("state=%v want=%v", m.state, StateDashboard)
	}
}
