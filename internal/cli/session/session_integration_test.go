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
			&cli.IntFlag{Name: "panes"},
			&cli.StringFlag{Name: "old"},
			&cli.StringFlag{Name: "new"},
			&cli.StringSliceFlag{Name: "env"},
		},
	}
}

type sessionFlow struct {
	t      *testing.T
	td     *testDaemon
	client *sessiond.Client
}

func newSessionFlow(t *testing.T) sessionFlow {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	td := newTestDaemon(t)
	client := dialTestClient(t, td)
	return sessionFlow{t: t, td: td, client: client}
}

func (f sessionFlow) deps() root.Dependencies {
	return root.Dependencies{Version: f.td.version, Connect: f.td.connect}
}

func (f sessionFlow) ctx(cmd *cli.Command, out io.Writer, json bool) root.CommandContext {
	return root.CommandContext{
		Context: context.Background(),
		Cmd:     cmd,
		Deps:    f.deps(),
		Out:     out,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
		JSON:    json,
	}
}

func (f sessionFlow) mustStart(name, path string, env ...string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("name", name)
	_ = cmd.Set("path", path)
	for _, kv := range env {
		_ = cmd.Set("env", kv)
	}
	if err := runStart(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runStart() error: %v", err)
	}
}

func (f sessionFlow) mustStartJSON(name, path string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("name", name)
	_ = cmd.Set("path", path)
	if err := runStart(f.ctx(cmd, io.Discard, true)); err != nil {
		f.t.Fatalf("runStart(json) error: %v", err)
	}
}

func (f sessionFlow) mustListContains(name string) {
	f.t.Helper()
	var out bytes.Buffer
	if err := runList(f.ctx(testCommand(), &out, false)); err != nil {
		f.t.Fatalf("runList() error: %v", err)
	}
	if !strings.Contains(out.String(), name) {
		f.t.Fatalf("runList output = %q", out.String())
	}
}

func (f sessionFlow) mustListJSONHasSessions() {
	f.t.Helper()
	var out bytes.Buffer
	if err := runList(f.ctx(testCommand(), &out, true)); err != nil {
		f.t.Fatalf("runList(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "\"sessions\"") {
		f.t.Fatalf("runList(json) output = %q", out.String())
	}
}

func (f sessionFlow) mustRename(oldName, newName string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("old", oldName)
	_ = cmd.Set("new", newName)
	if err := runRename(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runRename() error: %v", err)
	}
}

func (f sessionFlow) mustFocus(name string, json bool) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("name", name)
	if err := runFocus(f.ctx(cmd, io.Discard, json)); err != nil {
		f.t.Fatalf("runFocus(json=%v) error: %v", json, err)
	}
}

func (f sessionFlow) mustAssertFocusedSession(name string) {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := f.client.SnapshotState(ctx, 0)
	if err != nil {
		f.t.Fatalf("SnapshotState() error: %v", err)
	}
	if resp.FocusedSession != name {
		f.t.Fatalf("FocusedSession = %q", resp.FocusedSession)
	}
}

func (f sessionFlow) mustSnapshotContains(name string, json bool, want string) {
	f.t.Helper()
	var out bytes.Buffer
	if err := runSnapshot(f.ctx(testCommand(), &out, json)); err != nil {
		f.t.Fatalf("runSnapshot(json=%v) error: %v", json, err)
	}
	if !strings.Contains(out.String(), want) {
		f.t.Fatalf("runSnapshot output = %q", out.String())
	}
}

func (f sessionFlow) mustClose(name string, json bool) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("name", name)
	if err := runClose(f.ctx(cmd, io.Discard, json)); err != nil {
		f.t.Fatalf("runClose(json=%v) error: %v", json, err)
	}
}

func TestSessionCommandFlow(t *testing.T) {
	flow := newSessionFlow(t)
	path := t.TempDir()
	name := "sess"
	writeTestLayout(t, path, name)

	flow.mustStart(name, path, "FOO=bar")
	jsonPath := t.TempDir()
	jsonName := "sess-json"
	writeTestLayout(t, jsonPath, jsonName)
	flow.mustStartJSON(jsonName, jsonPath)

	waitForSessionSnapshot(t, flow.client, name)

	flow.mustListContains(name)
	flow.mustListJSONHasSessions()

	flow.mustRename(name, "renamed")
	name = "renamed"
	waitForSessionSnapshot(t, flow.client, name)

	flow.mustFocus(name, false)
	flow.mustFocus(name, true)
	flow.mustAssertFocusedSession(name)

	flow.mustSnapshotContains(name, false, name)
	flow.mustSnapshotContains(name, true, "\"snapshot\"")

	flow.mustClose(name, false)
	flow.mustClose(jsonName, true)
}

func TestSessionStartWithPanes(t *testing.T) {
	flow := newSessionFlow(t)
	path := t.TempDir()
	cmd := testCommand()
	_ = cmd.Set("name", "sess-panes")
	_ = cmd.Set("path", path)
	_ = cmd.Set("panes", "3")
	if err := runStart(flow.ctx(cmd, io.Discard, false)); err != nil {
		t.Fatalf("runStart(panes) error: %v", err)
	}
	snap := waitForSessionSnapshot(t, flow.client, "sess-panes")
	if len(snap.Panes) != 3 {
		t.Fatalf("expected 3 panes, got %d", len(snap.Panes))
	}
}
