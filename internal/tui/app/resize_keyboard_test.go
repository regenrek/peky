package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/sessiond"
)

func TestResizeModeBlockedByHardRaw(t *testing.T) {
	m := newTestModelLite()
	m.hardRaw = true
	msg := keyMsgFromTea(tea.KeyMsg{Type: tea.KeyCtrlR})
	_, handled := m.handleResizeKey(msg)
	if !handled {
		t.Fatalf("expected resize key handled")
	}
	if m.resize.mode {
		t.Fatalf("expected resize mode disabled while hard raw on")
	}
}

func TestResizeModeToggle(t *testing.T) {
	m := newTestModelLite()
	msg := keyMsgFromTea(tea.KeyMsg{Type: tea.KeyCtrlR})
	_, handled := m.handleResizeKey(msg)
	if !handled {
		t.Fatalf("expected resize key handled")
	}
	if !m.resize.mode {
		t.Fatalf("expected resize mode enabled")
	}
}

func TestResizeSnapToggleClearsState(t *testing.T) {
	m := newTestModelLite()
	m.enterResizeMode()
	m.resize.snap = true
	m.resize.key.snapState = sessiond.SnapState{Active: true, Target: 200}
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, handled := m.handleResizeModeKey(msg)
	if !handled {
		t.Fatalf("expected resize mode key handled")
	}
	if m.resize.snap {
		t.Fatalf("expected snap disabled after toggle")
	}
	if m.resize.key.snapState.Active || m.resize.key.snapState.Target != 0 {
		t.Fatalf("expected snap state cleared")
	}
}
