package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
	"github.com/regenrek/peakypanes/internal/layout"
	"github.com/regenrek/peakypanes/internal/native"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/sessiond/testkit"
)

type testDaemon struct {
	socket  string
	version string
	daemon  *sessiond.Daemon
}

func newTestDaemon(t *testing.T) *testDaemon {
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
	version := "test"
	daemon, err := sessiond.NewDaemon(sessiond.DaemonConfig{
		Version:       version,
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
	t.Cleanup(func() { _ = daemon.Stop() })
	return &testDaemon{socket: socket, version: version, daemon: daemon}
}

func (td *testDaemon) connect(ctx context.Context, version string) (*sessiond.Client, error) {
	return sessiond.Dial(ctx, td.socket, td.version)
}

func dialTestClient(t *testing.T, td *testDaemon) *sessiond.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := sessiond.Dial(ctx, td.socket, td.version)
	if err != nil {
		t.Fatalf("sessiond.Dial() error: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func writeTestLayout(t *testing.T, path, session string) {
	t.Helper()
	layoutCfg := &layout.LayoutConfig{
		Name: "test",
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
	if err := os.WriteFile(filepath.Join(path, ".peakypanes.yml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write .peakypanes.yml: %v", err)
	}
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

func testCommand() *cli.Command {
	return &cli.Command{
		Name: "session",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name"},
			&cli.StringFlag{Name: "path"},
			&cli.StringFlag{Name: "layout"},
			&cli.StringFlag{Name: "old"},
			&cli.StringFlag{Name: "new"},
			&cli.StringSliceFlag{Name: "env"},
		},
	}
}

func TestSessionCommandFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	td := newTestDaemon(t)
	client := dialTestClient(t, td)

	path := t.TempDir()
	name := "sess"
	writeTestLayout(t, path, name)

	startCmd := testCommand()
	_ = startCmd.Set("name", name)
	_ = startCmd.Set("path", path)
	_ = startCmd.Set("env", "FOO=bar")
	startCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     startCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runStart(startCtx); err != nil {
		t.Fatalf("runStart() error: %v", err)
	}
	jsonPath := t.TempDir()
	jsonName := "sess-json"
	writeTestLayout(t, jsonPath, jsonName)
	startJSONCmd := testCommand()
	_ = startJSONCmd.Set("name", jsonName)
	_ = startJSONCmd.Set("path", jsonPath)
	startJSON := root.CommandContext{
		Context: context.Background(),
		Cmd:     startJSONCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
		JSON:    true,
	}
	if err := runStart(startJSON); err != nil {
		t.Fatalf("runStart(json) error: %v", err)
	}

	waitForSessionSnapshot(t, client, name)

	var listOut bytes.Buffer
	listCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     testCommand(),
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     &listOut,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runList(listCtx); err != nil {
		t.Fatalf("runList() error: %v", err)
	}
	if !strings.Contains(listOut.String(), name) {
		t.Fatalf("runList output = %q", listOut.String())
	}
	var listJSON bytes.Buffer
	listJSONCtx := listCtx
	listJSONCtx.JSON = true
	listJSONCtx.Out = &listJSON
	if err := runList(listJSONCtx); err != nil {
		t.Fatalf("runList(json) error: %v", err)
	}
	if !strings.Contains(listJSON.String(), "\"sessions\"") {
		t.Fatalf("runList(json) output = %q", listJSON.String())
	}

	renameCmd := testCommand()
	_ = renameCmd.Set("old", name)
	_ = renameCmd.Set("new", "renamed")
	renameCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     renameCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runRename(renameCtx); err != nil {
		t.Fatalf("runRename() error: %v", err)
	}
	name = "renamed"
	waitForSessionSnapshot(t, client, name)

	focusCmd := testCommand()
	_ = focusCmd.Set("name", name)
	focusCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     focusCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runFocus(focusCtx); err != nil {
		t.Fatalf("runFocus() error: %v", err)
	}
	focusJSON := focusCtx
	focusJSON.JSON = true
	focusJSON.Out = io.Discard
	if err := runFocus(focusJSON); err != nil {
		t.Fatalf("runFocus(json) error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := client.SnapshotState(ctx, 0)
	if err != nil {
		t.Fatalf("SnapshotState() error: %v", err)
	}
	if resp.FocusedSession != name {
		t.Fatalf("FocusedSession = %q", resp.FocusedSession)
	}

	var snapOut bytes.Buffer
	snapshotCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     testCommand(),
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     &snapOut,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runSnapshot(snapshotCtx); err != nil {
		t.Fatalf("runSnapshot() error: %v", err)
	}
	if !strings.Contains(snapOut.String(), name) {
		t.Fatalf("runSnapshot output = %q", snapOut.String())
	}
	var snapJSON bytes.Buffer
	snapJSONCtx := snapshotCtx
	snapJSONCtx.JSON = true
	snapJSONCtx.Out = &snapJSON
	if err := runSnapshot(snapJSONCtx); err != nil {
		t.Fatalf("runSnapshot(json) error: %v", err)
	}
	if !strings.Contains(snapJSON.String(), "\"snapshot\"") {
		t.Fatalf("runSnapshot(json) output = %q", snapJSON.String())
	}

	killCmd := testCommand()
	_ = killCmd.Set("name", name)
	killCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     killCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runKill(killCtx); err != nil {
		t.Fatalf("runKill() error: %v", err)
	}
	killJSONCmd := testCommand()
	_ = killJSONCmd.Set("name", jsonName)
	killJSON := root.CommandContext{
		Context: context.Background(),
		Cmd:     killJSONCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
		JSON:    true,
	}
	if err := runKill(killJSON); err != nil {
		t.Fatalf("runKill(json) error: %v", err)
	}
}
