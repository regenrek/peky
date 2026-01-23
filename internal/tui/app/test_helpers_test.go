package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/sessiond/testkit"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	client, daemon := newTestDaemon(t)
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = daemon.Stop() })

	model, err := NewModel(client)
	if err != nil {
		t.Fatalf("NewModel() error: %v", err)
	}
	t.Cleanup(func() {
		if model.paneViewClient != nil {
			_ = model.paneViewClient.Close()
		}
	})
	model.width = 80
	model.height = 24
	model.state = StateDashboard
	if model.config == nil {
		model.config = &layout.Config{}
	}
	model.config.QuickReply.Enabled = true
	return model
}

func startNativeSession(t *testing.T, model *Model, name string) native.SessionSnapshot {
	t.Helper()
	if model == nil || model.client == nil {
		t.Fatalf("session client unavailable")
	}
	if name == "" {
		name = "sess"
	}
	path := t.TempDir()
	writeProjectLayout(t, path, name)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := model.client.StartSession(ctx, sessiond.StartSessionRequest{
		Name:       name,
		Path:       path,
		LayoutName: "",
	}); err != nil {
		t.Fatalf("StartSession() error: %v", err)
	}
	return waitForSessionSnapshot(t, model.client, name)
}

func newTestDaemon(t *testing.T) (*sessiond.Client, *sessiond.Daemon) {
	t.Helper()
	baseDir := t.TempDir()
	if runtime.GOOS != "windows" {
		if dir, err := os.MkdirTemp("/tmp", "ppd-"); err == nil {
			baseDir = dir
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
		}
	}
	socket := filepath.Join(baseDir, "daemon.sock")
	pid := filepath.Join(baseDir, "daemon.pid")
	daemon, err := sessiond.NewDaemon(sessiond.DaemonConfig{
		Version:       "test",
		SocketPath:    socket,
		PidPath:       pid,
		HandleSignals: false,
	})
	if err != nil {
		t.Fatalf("NewDaemon() error: %v", err)
	}
	if err := daemon.Start(); err != nil {
		t.Fatalf("daemon.Start() error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := sessiond.Dial(ctx, socket, "test")
	if err != nil {
		_ = daemon.Stop()
		t.Fatalf("sessiond.Dial() error: %v", err)
	}
	return client, daemon
}

func waitForSessionSnapshot(t *testing.T, client *sessiond.Client, name string) native.SessionSnapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	snap, err := testkit.WaitForSessionSnapshot(ctx, client, name)
	if err != nil {
		t.Fatalf("waitForSessionSnapshot() error: %v", err)
	}
	return snap
}

func sessionExists(t *testing.T, client *sessiond.Client, name string) bool {
	t.Helper()
	if client == nil {
		t.Fatalf("session client unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := client.SnapshotState(ctx, 0)
	if err != nil {
		t.Fatalf("SnapshotState() error: %v", err)
	}
	for _, snap := range resp.Sessions {
		if snap.Name == name {
			return true
		}
	}
	return false
}

func writeProjectLayout(t *testing.T, path, session string) {
	t.Helper()
	layoutCfg := &layout.LayoutConfig{
		Panes: []layout.PaneDef{{
			Title: "pane",
			Cmd:   "cat",
		}},
	}
	layoutYAML, err := layoutCfg.ToYAML()
	if err != nil {
		t.Fatalf("layout ToYAML error: %v", err)
	}
	content := fmt.Sprintf("session: %s\n\nlayout:\n", session)
	for _, line := range strings.Split(layoutYAML, "\n") {
		if line != "" {
			content += "  " + line + "\n"
		}
	}
	if err := os.WriteFile(filepath.Join(path, ".peky.yml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write .peky.yml: %v", err)
	}
}
