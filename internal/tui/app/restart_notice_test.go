package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/regenrek/peakypanes/internal/identity"
)

func TestServerStatusClickOpensRestartNotice(t *testing.T) {
	m := newTestModelLite()
	m.width = 120
	m.height = 40
	m.state = StateDashboard
	m.restartNoticePending = true

	rect, ok := m.serverStatusRect()
	if !ok {
		t.Fatalf("expected serverStatusRect ok")
	}
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      rect.X,
		Y:      rect.Y,
	}
	_, _ = m.updateDashboardMouse(msg)
	if m.state != StateRestartNotice {
		t.Fatalf("state=%v want=%v", m.state, StateRestartNotice)
	}
}

func TestDaemonReconnectSetsRestartNoticePending(t *testing.T) {
	m := newTestModelLite()
	m.daemonDisconnected = true

	client, daemon := newTestDaemon(t)
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = daemon.Stop() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	paneClient, err := client.Clone(ctx)
	if err != nil {
		t.Fatalf("Clone() pane client: %v", err)
	}
	defer func() { _ = paneClient.Close() }()

	_ = m.handleDaemonReconnect(daemonReconnectMsg{Client: client, PaneViewClient: paneClient})
	if !m.restartNoticePending {
		t.Fatalf("expected restartNoticePending=true")
	}
}

func TestRestartNoticeFlagActive(t *testing.T) {
	dir := t.TempDir()
	flag := filepath.Join(dir, identity.RestartNoticeFlagFile)
	if restartNoticeFlagActive(flag) {
		t.Fatalf("expected inactive for missing flag")
	}
	if err := os.WriteFile(flag, []byte("1\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !restartNoticeFlagActive(flag) {
		t.Fatalf("expected active for flag=1")
	}
	if err := os.WriteFile(flag, []byte("0\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if restartNoticeFlagActive(flag) {
		t.Fatalf("expected inactive for flag=0")
	}
}

func TestRestartNoticeFlagPathUsesConfigDir(t *testing.T) {
	m := newTestModelLite()
	m.configPath = filepath.Join(t.TempDir(), "config.yml")

	got := m.restartNoticeFlagPath()
	want := filepath.Join(filepath.Dir(m.configPath), identity.RestartNoticeFlagFile)
	if got != want {
		t.Fatalf("restartNoticeFlagPath() = %q, want %q", got, want)
	}
}
