package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/update"
)

func TestUpdateDialogLaterMarksPrompt(t *testing.T) {
	m := newTestModelLite()
	m.updatePolicy = update.DefaultPolicy()
	m.updateState = update.State{CurrentVersion: "1.0.0", LatestVersion: "1.1.0"}
	m.openUpdateDialog()
	if m.state != StateUpdateDialog {
		t.Fatalf("expected update dialog state")
	}
	m.updateUpdateDialog(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.state != StateDashboard {
		t.Fatalf("expected dashboard after later")
	}
	if m.updateState.LastPromptUnixMs == 0 {
		t.Fatalf("expected last prompt set")
	}
}

func TestUpdateDialogSkipSetsVersion(t *testing.T) {
	m := newTestModelLite()
	m.updatePolicy = update.DefaultPolicy()
	m.updateState = update.State{CurrentVersion: "1.0.0", LatestVersion: "1.2.0"}
	m.openUpdateDialog()
	m.updateUpdateDialog(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.updateState.SkippedVersion != "1.2.0" {
		t.Fatalf("expected skipped version set")
	}
}

func TestUpdateBannerInfo(t *testing.T) {
	m := newTestModelLite()
	m.updatePolicy = update.DefaultPolicy()
	m.updateState = update.State{CurrentVersion: "1.0.0", LatestVersion: "1.1.0"}
	label, hint, ok := m.updateBannerInfo()
	if !ok || label == "" || hint == "" {
		t.Fatalf("expected banner info")
	}
	m.updatePendingRestart = true
	label, _, ok = m.updateBannerInfo()
	if !ok || label == "" {
		t.Fatalf("expected restart banner info")
	}
}

func TestUpdateDialogRemindLabel(t *testing.T) {
	m := newTestModelLite()
	m.updatePolicy = update.Policy{PromptCooldown: 48 * time.Hour}
	view := m.updateDialogView()
	if view.RemindLabel != "Remind in 2 days" {
		t.Fatalf("expected remind label, got %q", view.RemindLabel)
	}
}
