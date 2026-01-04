package pane

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

func startTestSession(t *testing.T, client *sessiond.Client, name, path string) native.SessionSnapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.StartSession(ctx, sessiond.StartSessionRequest{
		Name:       name,
		Path:       path,
		LayoutName: "",
	}); err != nil {
		t.Fatalf("StartSession() error: %v", err)
	}
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()
	snap, err := testkit.WaitForSessionSnapshot(waitCtx, client, name)
	if err != nil {
		t.Fatalf("waitForSessionSnapshot() error: %v", err)
	}
	return snap
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
		Name: "pane",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "session"},
			&cli.StringFlag{Name: "pane-id"},
			&cli.StringFlag{Name: "orientation"},
			&cli.StringFlag{Name: "name"},
			&cli.StringFlag{Name: "text"},
			&cli.StringFlag{Name: "command"},
			&cli.StringFlag{Name: "scope"},
			&cli.StringFlag{Name: "mode"},
			&cli.StringFlag{Name: "action"},
			&cli.StringFlag{Name: "key"},
			&cli.StringFlag{Name: "signal"},
			&cli.StringFlag{Name: "grep"},
			&cli.StringFlag{Name: "since"},
			&cli.StringFlag{Name: "until"},
			&cli.IntFlag{Name: "index"},
			&cli.IntFlag{Name: "a"},
			&cli.IntFlag{Name: "b"},
			&cli.IntFlag{Name: "cols"},
			&cli.IntFlag{Name: "rows"},
			&cli.IntFlag{Name: "lines"},
			&cli.IntFlag{Name: "count"},
			&cli.IntFlag{Name: "delta-x"},
			&cli.IntFlag{Name: "delta-y"},
			&cli.IntFlag{Name: "limit"},
			&cli.DurationFlag{Name: "delay"},
			&cli.DurationFlag{Name: "submit-delay"},
			&cli.DurationFlag{Name: "timeout"},
			&cli.BoolFlag{Name: "confirm"},
			&cli.BoolFlag{Name: "require-ack"},
			&cli.BoolFlag{Name: "focus", Value: true},
			&cli.BoolFlag{Name: "follow"},
			&cli.BoolFlag{Name: "scrollback-toggle"},
			&cli.BoolFlag{Name: "copy-toggle"},
			&cli.StringSliceFlag{Name: "mods"},
			&cli.StringSliceFlag{Name: "tag"},
		},
	}
}

type paneFlow struct {
	t           *testing.T
	td          *testDaemon
	client      *sessiond.Client
	sessionName string
	paneID      string
	paneIndex   string
	otherPaneID string
	otherIndex  string
	swapIndexA  string
}

func newPaneFlow(t *testing.T) paneFlow {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	td := newTestDaemon(t)
	client := dialTestClient(t, td)
	path := t.TempDir()
	sessionName := "sess"
	writeTestLayout(t, path, sessionName)
	snap := startTestSession(t, client, sessionName, path)
	if len(snap.Panes) == 0 {
		t.Fatalf("session snapshot missing panes")
	}
	pane := snap.Panes[0]
	return paneFlow{
		t:           t,
		td:          td,
		client:      client,
		sessionName: sessionName,
		paneID:      pane.ID,
		paneIndex:   pane.Index,
	}
}

func (f paneFlow) deps() root.Dependencies {
	return root.Dependencies{Version: f.td.version, Connect: f.td.connect}
}

func (f paneFlow) ctx(cmd *cli.Command, out io.Writer, json bool) root.CommandContext {
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

func (f paneFlow) mustList() {
	f.t.Helper()
	cmd := testCommand()
	if err := cmd.Set("session", f.sessionName); err != nil {
		f.t.Fatalf("listCmd.Set(session) error: %v", err)
	}
	var out bytes.Buffer
	if err := runList(f.ctx(cmd, &out, false)); err != nil {
		f.t.Fatalf("runList() error: %v", err)
	}
	if !strings.Contains(out.String(), f.paneID) {
		f.t.Fatalf("runList output = %q", out.String())
	}
}

func (f paneFlow) mustListJSON() {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("session", f.sessionName)
	var out bytes.Buffer
	if err := runList(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runList(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "\"panes\"") {
		f.t.Fatalf("runList(json) output = %q", out.String())
	}
}

func (f paneFlow) mustRename(newName string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("session", f.sessionName)
	_ = cmd.Set("index", f.paneIndex)
	_ = cmd.Set("name", newName)
	if err := runRename(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runRename() error: %v", err)
	}
	updated := waitForSessionSnapshot(f.t, f.client, f.sessionName)
	if len(updated.Panes) == 0 || updated.Panes[0].Title != newName {
		f.t.Fatalf("pane rename missing: %#v", updated.Panes)
	}
}

func (f *paneFlow) mustSplitVerticalAndAssertFocus() {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("session", f.sessionName)
	_ = cmd.Set("index", f.paneIndex)
	_ = cmd.Set("orientation", "vertical")
	if err := runSplit(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runSplit() error: %v", err)
	}
	updated := waitForSessionSnapshot(f.t, f.client, f.sessionName)
	if len(updated.Panes) < 2 {
		f.t.Fatalf("expected split pane, got %#v", updated.Panes)
	}

	otherPaneID, ok := findOtherPaneID(updated, f.paneID)
	if !ok {
		f.t.Fatalf("expected new pane id after split")
	}
	f.otherPaneID = otherPaneID
	f.swapIndexA = findPaneIndexByID(updated, f.paneID)
	f.otherIndex = findPaneIndexByID(updated, otherPaneID)
	if f.swapIndexA == "" || f.otherIndex == "" {
		f.t.Fatalf("missing pane indices after split: swapIndexA=%q otherIndex=%q panes=%#v", f.swapIndexA, f.otherIndex, updated.Panes)
	}

	focusCheckCtx, focusCheckCancel := context.WithTimeout(context.Background(), 2*time.Second)
	focusSnap, err := f.client.SnapshotState(focusCheckCtx, 0)
	focusCheckCancel()
	if err != nil {
		f.t.Fatalf("SnapshotState() error: %v", err)
	}
	if focusSnap.FocusedPaneID != otherPaneID {
		f.t.Fatalf("expected focus %q, got %q", otherPaneID, focusSnap.FocusedPaneID)
	}
}

func findOtherPaneID(snap native.SessionSnapshot, excludeID string) (string, bool) {
	for _, p := range snap.Panes {
		if p.ID != excludeID {
			return p.ID, true
		}
	}
	return "", false
}

func findPaneIndexByID(snap native.SessionSnapshot, id string) string {
	for _, p := range snap.Panes {
		if p.ID == id {
			return p.Index
		}
	}
	return ""
}

func (f paneFlow) mustSendText(paneID, text string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("text", text)
	if err := runSend(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runSend() error: %v", err)
	}
	var out bytes.Buffer
	if err := runSend(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runSend(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "pane.send") {
		f.t.Fatalf("runSend(json) output = %q", out.String())
	}
}

func (f paneFlow) mustRunCommand(paneID, command string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("command", command)
	if err := runRun(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runRun() error: %v", err)
	}
}

func (f paneFlow) mustView(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("rows", "5")
	_ = cmd.Set("cols", "20")
	_ = cmd.Set("mode", "plain")
	if err := runView(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runView() error: %v", err)
	}
	var out bytes.Buffer
	if err := runView(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runView(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "\"pane_id\"") {
		f.t.Fatalf("runView(json) output = %q", out.String())
	}
}

func (f paneFlow) mustSnapshot(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("rows", "10")
	if err := runSnapshot(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runSnapshot() error: %v", err)
	}
	var out bytes.Buffer
	if err := runSnapshot(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runSnapshot(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "\"pane_id\"") {
		f.t.Fatalf("runSnapshot(json) output = %q", out.String())
	}
}

func (f paneFlow) mustHistory(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("limit", "5")
	if err := runHistory(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runHistory() error: %v", err)
	}
	var out bytes.Buffer
	if err := runHistory(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runHistory(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "\"entries\"") {
		f.t.Fatalf("runHistory(json) output = %q", out.String())
	}
}

func (f paneFlow) mustTagLifecycle(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("tag", "agent")
	if err := runTagAdd(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runTagAdd() error: %v", err)
	}
	if err := runTagList(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runTagList() error: %v", err)
	}
	var out bytes.Buffer
	if err := runTagList(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runTagList(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "\"tags\"") {
		f.t.Fatalf("runTagList(json) output = %q", out.String())
	}
	if err := runTagRemove(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runTagRemove() error: %v", err)
	}
}

func (f paneFlow) mustActionScrollUp(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("action", "scroll-up")
	_ = cmd.Set("lines", "1")
	if err := runAction(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runAction() error: %v", err)
	}
	var out bytes.Buffer
	if err := runAction(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runAction(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "pane.action") {
		f.t.Fatalf("runAction(json) output = %q", out.String())
	}
}

func (f paneFlow) mustKeyCtrlK(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("key", "k")
	_ = cmd.Set("mods", "ctrl")
	if err := runKey(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runKey() error: %v", err)
	}
	var out bytes.Buffer
	if err := runKey(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runKey(json) error: %v", err)
	}
	if !strings.Contains(out.String(), "pane.key") {
		f.t.Fatalf("runKey(json) output = %q", out.String())
	}
}

func (f paneFlow) mustTail(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("follow", "false")
	_ = cmd.Set("lines", "5")
	if err := runTail(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runTail() error: %v", err)
	}
	var out bytes.Buffer
	if err := runTail(f.ctx(cmd, &out, true)); err != nil {
		f.t.Fatalf("runTail(json) error: %v", err)
	}
	if out.Len() == 0 {
		f.t.Fatalf("runTail(json) empty output")
	}
}

func (f paneFlow) mustSwap() {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("session", f.sessionName)
	_ = cmd.Set("a", f.swapIndexA)
	_ = cmd.Set("b", f.otherIndex)
	if err := runSwap(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runSwap() error: %v", err)
	}
}

func (f paneFlow) mustResize(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	_ = cmd.Set("cols", "100")
	_ = cmd.Set("rows", "40")
	if err := runResize(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runResize() error: %v", err)
	}
}

func (f paneFlow) mustFocus(paneID string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", paneID)
	if err := runFocus(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runFocus() error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	focusSnap, err := f.client.SnapshotState(ctx, 0)
	if err != nil {
		f.t.Fatalf("SnapshotState() error: %v", err)
	}
	if focusSnap.FocusedPaneID != paneID {
		f.t.Fatalf("FocusedPaneID = %q", focusSnap.FocusedPaneID)
	}
}

func (f paneFlow) mustSendFocused(text string) {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", "@focused")
	_ = cmd.Set("text", text)
	if err := runSend(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runSend(@focused) error: %v", err)
	}
}

func (f paneFlow) mustCloseOtherPane() {
	f.t.Helper()
	cmd := testCommand()
	_ = cmd.Set("pane-id", f.otherPaneID)
	if err := runClose(f.ctx(cmd, io.Discard, false)); err != nil {
		f.t.Fatalf("runClose() error: %v", err)
	}
	updated := waitForSessionSnapshot(f.t, f.client, f.sessionName)
	for _, p := range updated.Panes {
		if p.ID == f.otherPaneID {
			f.t.Fatalf("expected pane %q to be closed", f.otherPaneID)
		}
	}
}

func TestPaneCommandFlow(t *testing.T) {
	flow := newPaneFlow(t)
	flow.mustList()
	flow.mustListJSON()
	flow.mustRename("renamed")
	flow.mustSplitVerticalAndAssertFocus()
	flow.mustSendText(flow.paneID, "hello")
	flow.mustRunCommand(flow.paneID, "echo hi")
	flow.mustView(flow.paneID)
	flow.mustSnapshot(flow.paneID)
	flow.mustHistory(flow.paneID)
	flow.mustTagLifecycle(flow.paneID)
	flow.mustActionScrollUp(flow.paneID)
	flow.mustKeyCtrlK(flow.paneID)
	flow.mustTail(flow.paneID)
	flow.mustSwap()
	flow.mustResize(flow.paneID)
	flow.mustFocus(flow.paneID)
	flow.mustSendFocused("ping")
	flow.mustCloseOtherPane()
	if _, err := strconv.Atoi(flow.paneIndex); err != nil {
		t.Fatalf("pane index not numeric: %q", flow.paneIndex)
	}
}

func TestFocusedPaneTokenCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	td := newTestDaemon(t)
	client := dialTestClient(t, td)

	type testCase struct {
		name  string
		run   func(root.CommandContext) error
		setup func(cmd *cli.Command)
	}

	cases := []testCase{
		{
			name: "rename",
			run:  runRename,
			setup: func(cmd *cli.Command) {
				_ = cmd.Set("pane-id", "@focused")
				_ = cmd.Set("name", "renamed")
			},
		},
		{
			name: "send",
			run:  runSend,
			setup: func(cmd *cli.Command) {
				_ = cmd.Set("pane-id", "@focused")
				_ = cmd.Set("text", "ping")
			},
		},
		{
			name: "run",
			run:  runRun,
			setup: func(cmd *cli.Command) {
				_ = cmd.Set("pane-id", "@focused")
				_ = cmd.Set("command", "echo hello")
			},
		},
		{
			name: "add",
			run:  runAdd,
			setup: func(cmd *cli.Command) {
				_ = cmd.Set("pane-id", "@focused")
			},
		},
		{
			name: "close",
			run:  runClose,
			setup: func(cmd *cli.Command) {
				_ = cmd.Set("pane-id", "@focused")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			path := t.TempDir()
			sessionName := fmt.Sprintf("sess-%s", tc.name)
			writeTestLayout(t, path, sessionName)
			snap := startTestSession(t, client, sessionName, path)
			if len(snap.Panes) == 0 {
				t.Fatalf("session snapshot missing panes")
			}
			paneID := snap.Panes[0].ID

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if err := client.FocusSession(ctx, sessionName); err != nil {
				cancel()
				t.Fatalf("FocusSession() error: %v", err)
			}
			if err := client.FocusPane(ctx, paneID); err != nil {
				cancel()
				t.Fatalf("FocusPane() error: %v", err)
			}
			cancel()

			cmd := testCommand()
			tc.setup(cmd)
			var out bytes.Buffer
			cmdCtx := root.CommandContext{
				Context: context.Background(),
				Cmd:     cmd,
				Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
				Out:     &out,
				ErrOut:  io.Discard,
				Stdin:   strings.NewReader(""),
			}
			if err := tc.run(cmdCtx); err != nil {
				t.Fatalf("%s(@focused) error: %v", tc.name, err)
			}
		})
	}
}

func TestPaneAddFocusesNewPane(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	td := newTestDaemon(t)
	client := dialTestClient(t, td)
	path := t.TempDir()
	sessionName := "sess-add"
	writeTestLayout(t, path, sessionName)
	snap := startTestSession(t, client, sessionName, path)
	if len(snap.Panes) == 0 {
		t.Fatalf("session snapshot missing panes")
	}
	pane := snap.Panes[0]

	addCmd := testCommand()
	_ = addCmd.Set("session", sessionName)
	_ = addCmd.Set("index", pane.Index)
	addCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     addCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runAdd(addCtx); err != nil {
		t.Fatalf("runAdd() error: %v", err)
	}

	updated := waitForSessionSnapshot(t, client, sessionName)
	if len(updated.Panes) < 2 {
		t.Fatalf("expected added pane, got %#v", updated.Panes)
	}
	newPaneID := ""
	for _, p := range updated.Panes {
		if p.ID != pane.ID {
			newPaneID = p.ID
			break
		}
	}
	if newPaneID == "" {
		t.Fatalf("expected new pane id after add")
	}

	focusCtx, focusCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer focusCancel()
	focusSnap, err := client.SnapshotState(focusCtx, 0)
	if err != nil {
		t.Fatalf("SnapshotState() error: %v", err)
	}
	if focusSnap.FocusedPaneID != newPaneID {
		t.Fatalf("expected focus %q, got %q", newPaneID, focusSnap.FocusedPaneID)
	}
}
