package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRestartDaemonCmdErrorFlow(t *testing.T) {
	m := newTestModelLite()
	m.setState(StateConfirmRestart)

	_, cmd := m.updateConfirmRestart(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected restart cmd")
	}
	msg, ok := cmd().(daemonRestartMsg)
	if !ok || msg.Err == nil {
		t.Fatalf("expected daemonRestartMsg error, got %#v", msg)
	}
	_ = m.handleDaemonRestart(msg)
	if m.toast.Level != toastError || !strings.Contains(m.toast.Text, "Restart failed") {
		t.Fatalf("toast=%#v", m.toast)
	}
}

func TestStopDaemonCmdErrorFlow(t *testing.T) {
	m := newTestModelLite()
	m.pendingQuit = quitActionStop

	cmd := m.stopDaemonCmd()
	if cmd == nil {
		t.Fatalf("expected stop cmd")
	}
	msg, ok := cmd().(daemonStopMsg)
	if !ok || msg.Err == nil {
		t.Fatalf("expected daemonStopMsg error, got %#v", msg)
	}
	_ = m.handleDaemonStop(msg)
	if m.toast.Level != toastError || !strings.Contains(m.toast.Text, "Stop daemon failed") {
		t.Fatalf("toast=%#v", m.toast)
	}
}
