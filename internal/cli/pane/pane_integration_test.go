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

func TestPaneCommandFlow(t *testing.T) {
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

	listCmd := testCommand()
	if err := listCmd.Set("session", sessionName); err != nil {
		t.Fatalf("listCmd.Set(session) error: %v", err)
	}
	var listOut bytes.Buffer
	listCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     listCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     &listOut,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runList(listCtx); err != nil {
		t.Fatalf("runList() error: %v", err)
	}
	if !strings.Contains(listOut.String(), pane.ID) {
		t.Fatalf("runList output = %q", listOut.String())
	}
	var listJSON bytes.Buffer
	listJSONCtx := listCtx
	listJSONCtx.JSON = true
	listJSONCtx.Out = &listJSON
	if err := runList(listJSONCtx); err != nil {
		t.Fatalf("runList(json) error: %v", err)
	}
	if !strings.Contains(listJSON.String(), "\"panes\"") {
		t.Fatalf("runList(json) output = %q", listJSON.String())
	}

	renameCmd := testCommand()
	_ = renameCmd.Set("session", sessionName)
	_ = renameCmd.Set("index", pane.Index)
	_ = renameCmd.Set("name", "renamed")
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

	updated := waitForSessionSnapshot(t, client, sessionName)
	if len(updated.Panes) == 0 || updated.Panes[0].Title != "renamed" {
		t.Fatalf("pane rename missing: %#v", updated.Panes)
	}

	splitCmd := testCommand()
	_ = splitCmd.Set("session", sessionName)
	_ = splitCmd.Set("index", pane.Index)
	_ = splitCmd.Set("orientation", "vertical")
	splitCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     splitCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runSplit(splitCtx); err != nil {
		t.Fatalf("runSplit() error: %v", err)
	}

	updated = waitForSessionSnapshot(t, client, sessionName)
	if len(updated.Panes) < 2 {
		t.Fatalf("expected split pane, got %#v", updated.Panes)
	}
	otherPane := updated.Panes[1]
	newPaneID := ""
	for _, p := range updated.Panes {
		if p.ID != pane.ID {
			newPaneID = p.ID
			break
		}
	}
	if newPaneID == "" {
		t.Fatalf("expected new pane id after split")
	}
	{
		focusCheckCtx, focusCheckCancel := context.WithTimeout(context.Background(), 2*time.Second)
		focusSnap, err := client.SnapshotState(focusCheckCtx, 0)
		focusCheckCancel()
		if err != nil {
			t.Fatalf("SnapshotState() error: %v", err)
		}
		if focusSnap.FocusedPaneID != newPaneID {
			t.Fatalf("expected focus %q, got %q", newPaneID, focusSnap.FocusedPaneID)
		}
	}

	sendCmd := testCommand()
	_ = sendCmd.Set("pane-id", pane.ID)
	_ = sendCmd.Set("text", "hello")
	sendCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     sendCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runSend(sendCtx); err != nil {
		t.Fatalf("runSend() error: %v", err)
	}
	var sendJSON bytes.Buffer
	sendJSONCtx := sendCtx
	sendJSONCtx.JSON = true
	sendJSONCtx.Out = &sendJSON
	if err := runSend(sendJSONCtx); err != nil {
		t.Fatalf("runSend(json) error: %v", err)
	}
	if !strings.Contains(sendJSON.String(), "pane.send") {
		t.Fatalf("runSend(json) output = %q", sendJSON.String())
	}

	runCmd := testCommand()
	_ = runCmd.Set("pane-id", pane.ID)
	_ = runCmd.Set("command", "echo hi")
	runCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     runCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runRun(runCtx); err != nil {
		t.Fatalf("runRun() error: %v", err)
	}

	viewCmd := testCommand()
	_ = viewCmd.Set("pane-id", pane.ID)
	_ = viewCmd.Set("rows", "5")
	_ = viewCmd.Set("cols", "20")
	_ = viewCmd.Set("mode", "plain")
	viewCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     viewCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runView(viewCtx); err != nil {
		t.Fatalf("runView() error: %v", err)
	}
	var viewJSON bytes.Buffer
	viewJSONCtx := viewCtx
	viewJSONCtx.JSON = true
	viewJSONCtx.Out = &viewJSON
	if err := runView(viewJSONCtx); err != nil {
		t.Fatalf("runView(json) error: %v", err)
	}
	if !strings.Contains(viewJSON.String(), "\"pane_id\"") {
		t.Fatalf("runView(json) output = %q", viewJSON.String())
	}

	snapshotCmd := testCommand()
	_ = snapshotCmd.Set("pane-id", pane.ID)
	_ = snapshotCmd.Set("rows", "10")
	snapshotCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     snapshotCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runSnapshot(snapshotCtx); err != nil {
		t.Fatalf("runSnapshot() error: %v", err)
	}
	var snapJSON bytes.Buffer
	snapJSONCtx := snapshotCtx
	snapJSONCtx.JSON = true
	snapJSONCtx.Out = &snapJSON
	if err := runSnapshot(snapJSONCtx); err != nil {
		t.Fatalf("runSnapshot(json) error: %v", err)
	}
	if !strings.Contains(snapJSON.String(), "\"pane_id\"") {
		t.Fatalf("runSnapshot(json) output = %q", snapJSON.String())
	}

	historyCmd := testCommand()
	_ = historyCmd.Set("pane-id", pane.ID)
	_ = historyCmd.Set("limit", "5")
	historyCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     historyCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runHistory(historyCtx); err != nil {
		t.Fatalf("runHistory() error: %v", err)
	}
	var historyJSON bytes.Buffer
	historyJSONCtx := historyCtx
	historyJSONCtx.JSON = true
	historyJSONCtx.Out = &historyJSON
	if err := runHistory(historyJSONCtx); err != nil {
		t.Fatalf("runHistory(json) error: %v", err)
	}
	if !strings.Contains(historyJSON.String(), "\"entries\"") {
		t.Fatalf("runHistory(json) output = %q", historyJSON.String())
	}

	tagCmd := testCommand()
	_ = tagCmd.Set("pane-id", pane.ID)
	_ = tagCmd.Set("tag", "agent")
	tagCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     tagCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runTagAdd(tagCtx); err != nil {
		t.Fatalf("runTagAdd() error: %v", err)
	}
	if err := runTagList(tagCtx); err != nil {
		t.Fatalf("runTagList() error: %v", err)
	}
	var tagJSON bytes.Buffer
	tagJSONCtx := tagCtx
	tagJSONCtx.JSON = true
	tagJSONCtx.Out = &tagJSON
	if err := runTagList(tagJSONCtx); err != nil {
		t.Fatalf("runTagList(json) error: %v", err)
	}
	if !strings.Contains(tagJSON.String(), "\"tags\"") {
		t.Fatalf("runTagList(json) output = %q", tagJSON.String())
	}
	if err := runTagRemove(tagCtx); err != nil {
		t.Fatalf("runTagRemove() error: %v", err)
	}

	actionCmd := testCommand()
	_ = actionCmd.Set("pane-id", pane.ID)
	_ = actionCmd.Set("action", "scroll-up")
	_ = actionCmd.Set("lines", "1")
	actionCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     actionCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runAction(actionCtx); err != nil {
		t.Fatalf("runAction() error: %v", err)
	}
	var actionJSON bytes.Buffer
	actionJSONCtx := actionCtx
	actionJSONCtx.JSON = true
	actionJSONCtx.Out = &actionJSON
	if err := runAction(actionJSONCtx); err != nil {
		t.Fatalf("runAction(json) error: %v", err)
	}
	if !strings.Contains(actionJSON.String(), "pane.action") {
		t.Fatalf("runAction(json) output = %q", actionJSON.String())
	}

	keyCmd := testCommand()
	_ = keyCmd.Set("pane-id", pane.ID)
	_ = keyCmd.Set("key", "k")
	_ = keyCmd.Set("mods", "ctrl")
	keyCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     keyCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runKey(keyCtx); err != nil {
		t.Fatalf("runKey() error: %v", err)
	}
	var keyJSON bytes.Buffer
	keyJSONCtx := keyCtx
	keyJSONCtx.JSON = true
	keyJSONCtx.Out = &keyJSON
	if err := runKey(keyJSONCtx); err != nil {
		t.Fatalf("runKey(json) error: %v", err)
	}
	if !strings.Contains(keyJSON.String(), "pane.key") {
		t.Fatalf("runKey(json) output = %q", keyJSON.String())
	}

	tailCmd := testCommand()
	_ = tailCmd.Set("pane-id", pane.ID)
	_ = tailCmd.Set("follow", "false")
	_ = tailCmd.Set("lines", "5")
	tailCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     tailCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runTail(tailCtx); err != nil {
		t.Fatalf("runTail() error: %v", err)
	}
	var tailJSON bytes.Buffer
	tailJSONCtx := tailCtx
	tailJSONCtx.JSON = true
	tailJSONCtx.Out = &tailJSON
	if err := runTail(tailJSONCtx); err != nil {
		t.Fatalf("runTail(json) error: %v", err)
	}
	if tailJSON.Len() == 0 {
		t.Fatalf("runTail(json) empty output")
	}

	swapCmd := testCommand()
	_ = swapCmd.Set("session", sessionName)
	_ = swapCmd.Set("a", updated.Panes[0].Index)
	_ = swapCmd.Set("b", otherPane.Index)
	swapCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     swapCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runSwap(swapCtx); err != nil {
		t.Fatalf("runSwap() error: %v", err)
	}

	resizeCmd := testCommand()
	_ = resizeCmd.Set("pane-id", pane.ID)
	_ = resizeCmd.Set("cols", "100")
	_ = resizeCmd.Set("rows", "40")
	resizeCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     resizeCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runResize(resizeCtx); err != nil {
		t.Fatalf("runResize() error: %v", err)
	}

	focusCmd := testCommand()
	_ = focusCmd.Set("pane-id", pane.ID)
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	focusSnap, err := client.SnapshotState(ctx, 0)
	if err != nil {
		t.Fatalf("SnapshotState() error: %v", err)
	}
	if focusSnap.FocusedPaneID != pane.ID {
		t.Fatalf("FocusedPaneID = %q", focusSnap.FocusedPaneID)
	}

	focusedSendCmd := testCommand()
	_ = focusedSendCmd.Set("pane-id", "@focused")
	_ = focusedSendCmd.Set("text", "ping")
	focusedSendCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     focusedSendCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runSend(focusedSendCtx); err != nil {
		t.Fatalf("runSend(@focused) error: %v", err)
	}

	closeCmd := testCommand()
	_ = closeCmd.Set("pane-id", otherPane.ID)
	closeCtx := root.CommandContext{
		Context: context.Background(),
		Cmd:     closeCmd,
		Deps:    root.Dependencies{Version: td.version, Connect: td.connect},
		Out:     io.Discard,
		ErrOut:  io.Discard,
		Stdin:   strings.NewReader(""),
	}
	if err := runClose(closeCtx); err != nil {
		t.Fatalf("runClose() error: %v", err)
	}
	updated = waitForSessionSnapshot(t, client, sessionName)
	for _, p := range updated.Panes {
		if p.ID == otherPane.ID {
			t.Fatalf("expected pane %q to be closed", otherPane.ID)
		}
	}

	if _, err := strconv.Atoi(pane.Index); err != nil {
		t.Fatalf("pane index not numeric: %q", pane.Index)
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
