package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/skills"
)

func TestSkillsTargetPickerSelection(t *testing.T) {
	m := newTestModelLite()
	m.openSkillsInstall()

	if m.state != StateSkillsTargetPicker {
		t.Fatalf("state=%v want=%v", m.state, StateSkillsTargetPicker)
	}
	if len(m.skillsTargetItems) == 0 {
		t.Fatalf("expected skill targets")
	}

	m.skillsTargetPicker.Select(0)
	selected := m.skillsTargetItems[0].Selected
	m.updateSkillsTargetPicker(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if m.skillsTargetItems[0].Selected == selected {
		t.Fatalf("expected toggle selection")
	}
}

func TestSkillsTargetPickerEnter(t *testing.T) {
	m := newTestModelLite()
	m.openSkillsInstall()

	for _, item := range m.skillsTargetItems {
		item.Selected = false
	}

	m.updateSkillsTargetPicker(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateSkillsTargetPicker {
		t.Fatalf("state=%v want=%v", m.state, StateSkillsTargetPicker)
	}
	if !strings.Contains(m.toast.Text, "Select at least one target") {
		t.Fatalf("unexpected toast: %s", m.toast.Text)
	}

	m.skillsTargetItems[0].Selected = true
	_, cmd := m.updateSkillsTargetPicker(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateDashboard {
		t.Fatalf("state=%v want=%v", m.state, StateDashboard)
	}
	if cmd == nil {
		t.Fatalf("expected install cmd")
	}
	if !strings.Contains(m.toast.Text, "Installing skills") {
		t.Fatalf("unexpected toast: %s", m.toast.Text)
	}
}

func TestSkillsInstallResultToast(t *testing.T) {
	m := newTestModelLite()
	m.handleSkillsInstallResult(skillsInstallResultMsg{
		Result: skills.InstallResult{
			Records: []skills.InstallRecord{
				{SkillID: "peky-peky", Target: skills.TargetCodex, Status: "installed"},
			},
		},
	})
	if !strings.Contains(m.toast.Text, "Installed") {
		t.Fatalf("unexpected toast: %s", m.toast.Text)
	}
}
