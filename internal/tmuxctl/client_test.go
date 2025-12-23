package tmuxctl

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestSplitTmuxArgs(t *testing.T) {
	cases := []struct {
		input   string
		want    []string
		wantErr bool
	}{
		{input: "run-shell \"echo hi\"", want: []string{"run-shell", "echo hi"}},
		{input: "send-keys -t main \"go test ./...\"", want: []string{"send-keys", "-t", "main", "go test ./..."}},
		{input: "set -g status-style 'bg=black fg=white'", want: []string{"set", "-g", "status-style", "bg=black fg=white"}},
		{input: "", wantErr: true},
		{input: "unterminated \"quote", wantErr: true},
		{input: "trailing\\", wantErr: true},
	}

	for _, tt := range cases {
		t.Run(tt.input, func(t *testing.T) {
			got, err := splitTmuxArgs(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("splitTmuxArgs(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("splitTmuxArgs(%q) error: %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitTmuxArgs(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	if insideTmux() {
		t.Fatalf("insideTmux() should be false")
	}
	t.Setenv("TMUX", "/tmp/tmux")
	if !insideTmux() {
		t.Fatalf("insideTmux() should be true when TMUX is set")
	}
}

func TestNewClientProvidedPath(t *testing.T) {
	client, err := NewClient("/usr/local/bin/tmux")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	if client.Binary() != "/usr/local/bin/tmux" {
		t.Fatalf("Binary() = %q", client.Binary())
	}
}

func TestListSessionsNoServer(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"list-sessions", "-F", "#{session_name}"},
		stdout: "no server running",
		exit:   1,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("ListSessions() = %#v", sessions)
	}
	runner.assertDone()
}

func TestListSessionsParsesNames(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"list-sessions", "-F", "#{session_name}"},
		stdout: "one\n two\n\n",
		exit:   0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if !reflect.DeepEqual(sessions, []string{"one", "two"}) {
		t.Fatalf("ListSessions() = %#v", sessions)
	}
	runner.assertDone()
}

func TestCurrentSessionNoServer(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"display-message", "-p", "#S"},
		stdout: "no server running",
		exit:   1,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	session, err := client.CurrentSession(context.Background())
	if err != nil {
		t.Fatalf("CurrentSession() error: %v", err)
	}
	if session != "" {
		t.Fatalf("CurrentSession() = %q", session)
	}
	runner.assertDone()
}

func TestEnsureSessionCreatesGrid(t *testing.T) {
	tmpDir := t.TempDir()
	absDir, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("Abs() error: %v", err)
	}

	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{name: "tmux", args: []string{"has-session", "-t", "demo"}, exit: 1},
		{name: "tmux", args: []string{"new-session", "-d", "-s", "demo", "-c", absDir, "-P", "-F", "#{pane_id}"}, stdout: "%1\n", exit: 0},
		{name: "tmux", args: []string{"set-option", "-t", "demo", "remain-on-exit", "off"}, exit: 0},
		{name: "tmux", args: []string{"split-window", "-v", "-t", "%1", "-c", absDir, "-P", "-F", "#{pane_id}"}, stdout: "%2\n", exit: 0},
		{name: "tmux", args: []string{"split-window", "-h", "-t", "%1", "-c", absDir, "-P", "-F", "#{pane_id}"}, stdout: "%3\n", exit: 0},
		{name: "tmux", args: []string{"split-window", "-h", "-t", "%2", "-c", absDir, "-P", "-F", "#{pane_id}"}, stdout: "%4\n", exit: 0},
		{name: "tmux", args: []string{"select-layout", "-t", "demo", "tiled"}, exit: 0},
	}}

	client := &Client{bin: "tmux", run: runner.run}
	res, err := client.EnsureSession(context.Background(), Options{
		Session:  "demo",
		Layout:   layout.Grid{Rows: 2, Columns: 2},
		StartDir: absDir,
		Attach:   false,
	})
	if err != nil {
		t.Fatalf("EnsureSession() error: %v", err)
	}
	if !res.Created || res.Attached {
		t.Fatalf("EnsureSession() result = %#v", res)
	}
	runner.assertDone()
}

func TestEnsureSessionAttachExisting(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{name: "tmux", args: []string{"has-session", "-t", "demo"}, exit: 0},
		{name: "tmux", args: []string{"attach-session", "-t", "demo"}, exit: 0},
	}}
	client := &Client{bin: "tmux", run: runner.run}
	res, err := client.EnsureSession(context.Background(), Options{
		Session:  "demo",
		Layout:   layout.Default,
		StartDir: t.TempDir(),
		Attach:   true,
	})
	if err != nil {
		t.Fatalf("EnsureSession() error: %v", err)
	}
	if res.Created || !res.Attached {
		t.Fatalf("EnsureSession() result = %#v", res)
	}
	runner.assertDone()
}

func TestEnsureSessionValidationErrors(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}

	if _, err := client.EnsureSession(context.Background(), Options{}); err == nil {
		t.Fatalf("EnsureSession() expected error for empty session")
	}
	if _, err := client.EnsureSession(context.Background(), Options{Session: "demo", Layout: layout.Grid{Rows: 0, Columns: 2}}); err == nil {
		t.Fatalf("EnsureSession() expected error for invalid layout")
	}
	if _, err := client.EnsureSession(context.Background(), Options{Session: "demo", StartDir: filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatalf("EnsureSession() expected error for missing dir")
	}
}

func TestKillWindowNotFoundIgnored(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"kill-window", "-t", "demo:missing"},
		stdout: "can't find window",
		exit:   1,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.KillWindow(context.Background(), "demo", "missing"); err != nil {
		t.Fatalf("KillWindow() error: %v", err)
	}
	runner.assertDone()
}

func TestBindKey(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"bind-key", "C-a", "run-shell", "echo hi"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.BindKey(context.Background(), "C-a", "run-shell \"echo hi\""); err != nil {
		t.Fatalf("BindKey() error: %v", err)
	}
	runner.assertDone()
}

func TestBindKeyValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.BindKey(context.Background(), "", "run-shell ls"); err == nil {
		t.Fatalf("BindKey() expected error for empty key")
	}
	if err := client.BindKey(context.Background(), "C-a", ""); err == nil {
		t.Fatalf("BindKey() expected error for empty action")
	}
}

func TestSupportsPopup(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"list-commands"},
		stdout: "display-popup\nother",
		exit:   0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if !client.SupportsPopup(context.Background()) {
		t.Fatalf("SupportsPopup() should be true")
	}
	runner.assertDone()
}

func TestDisplayPopupArgs(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"display-popup", "-E", "-w", "80", "-h", "20", "-d", "/tmp", "bash", "-lc", "echo hi"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	err := client.DisplayPopup(context.Background(), PopupOptions{Width: "80", Height: "20", StartDir: "/tmp"}, []string{"bash", "-lc", "echo hi"})
	if err != nil {
		t.Fatalf("DisplayPopup() error: %v", err)
	}
	runner.assertDone()
}

func TestDisplayPopupValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.DisplayPopup(context.Background(), PopupOptions{}, nil); err == nil {
		t.Fatalf("DisplayPopup() expected error for empty command")
	}
}

func TestAttachExistingMissing(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"has-session", "-t", "missing"},
		exit: 1,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.AttachExisting(context.Background(), "missing"); err == nil {
		t.Fatalf("AttachExisting() expected error")
	}
	runner.assertDone()
}

func TestWrapTmuxErrUsesStderr(t *testing.T) {
	cmd := helperCmd(context.Background(), "", "stderr message", 1)
	_, err := cmd.Output()
	if err == nil {
		t.Fatalf("helper cmd should fail")
	}

	wrapped := wrapTmuxErr("test", err, nil)
	if !strings.Contains(wrapped.Error(), "stderr message") {
		t.Fatalf("wrapTmuxErr() = %q", wrapped.Error())
	}
}

func TestWrapTmuxErrUsesCombined(t *testing.T) {
	combined := []byte("combined output")
	wrapped := wrapTmuxErr("test", errors.New("boom"), combined)
	if !strings.Contains(wrapped.Error(), "combined output") {
		t.Fatalf("wrapTmuxErr() = %q", wrapped.Error())
	}
}

func TestSendKeysValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.SendKeys(context.Background(), ""); err == nil {
		t.Fatalf("SendKeys() expected error")
	}
}

func TestSendKeysLiteralArgs(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"send-keys", "-t", "%1", "-l", "hi there"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.SendKeysLiteral(context.Background(), "%1", "hi there"); err != nil {
		t.Fatalf("SendKeysLiteral() error: %v", err)
	}
	runner.assertDone()
}

func TestSelectPaneValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.SelectPane(context.Background(), "", "title"); err == nil {
		t.Fatalf("SelectPane() expected error")
	}
}

func TestSetOptionGlobal(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"set-option", "-g", "status", "on"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.SetOption(context.Background(), "-g", "status", "on"); err != nil {
		t.Fatalf("SetOption() error: %v", err)
	}
	runner.assertDone()
}

func TestSplitWindowWithCmdArgs(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"split-window", "-v", "-t", "%1", "-P", "-F", "#{pane_id}", "-c", "/tmp", "-p", "50", "echo hi"},
		stdout: "%2\n",
		exit:   0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	pane, err := client.SplitWindowWithCmd(context.Background(), "%1", "/tmp", true, 50, "echo hi", false)
	if err != nil {
		t.Fatalf("SplitWindowWithCmd() error: %v", err)
	}
	if pane != "%2" {
		t.Fatalf("SplitWindowWithCmd() pane = %q", pane)
	}
	runner.assertDone()
}

func TestNewSessionWithCmdArgs(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"new-session", "-d", "-s", "demo", "-P", "-F", "#{pane_id}", "-n", "win", "-c", "/tmp", "echo hi"},
		stdout: "%1\n",
		exit:   0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	pane, err := client.NewSessionWithCmd(context.Background(), "demo", "/tmp", "win", "echo hi")
	if err != nil {
		t.Fatalf("NewSessionWithCmd() error: %v", err)
	}
	if pane != "%1" {
		t.Fatalf("NewSessionWithCmd() pane = %q", pane)
	}
	runner.assertDone()
}

func TestNewWindowWithCmdArgs(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name:   "tmux",
		args:   []string{"new-window", "-t", "demo", "-P", "-F", "#{pane_id}", "-n", "win", "-c", "/tmp", "echo hi"},
		stdout: "%9\n",
		exit:   0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	pane, err := client.NewWindowWithCmd(context.Background(), "demo", "win", "/tmp", "echo hi", false)
	if err != nil {
		t.Fatalf("NewWindowWithCmd() error: %v", err)
	}
	if pane != "%9" {
		t.Fatalf("NewWindowWithCmd() pane = %q", pane)
	}
	runner.assertDone()
}

func TestRenameWindowValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.RenameWindow(context.Background(), "", "0", "x"); err == nil {
		t.Fatalf("RenameWindow() expected error for empty session")
	}
	if err := client.RenameWindow(context.Background(), "s", "", "x"); err == nil {
		t.Fatalf("RenameWindow() expected error for empty target")
	}
	if err := client.RenameWindow(context.Background(), "s", "0", ""); err == nil {
		t.Fatalf("RenameWindow() expected error for empty name")
	}
}

func TestSelectLayoutValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.SelectLayout(context.Background(), "", "tiled"); err == nil {
		t.Fatalf("SelectLayout() expected error")
	}
}

func TestSplitWindowValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.SplitWindow(context.Background(), "", "/tmp", false, 0); err == nil {
		t.Fatalf("SplitWindow() expected error")
	}
}

func TestNewWindowValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.NewWindow(context.Background(), "", "", "", ""); err == nil {
		t.Fatalf("NewWindow() expected error")
	}
}

func TestSessionTimeoutContext(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"has-session", "-t", "demo"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	_, err := client.EnsureSession(context.Background(), Options{
		Session: "demo",
		Timeout: 5 * time.Second,
		Attach:  false,
	})
	if err != nil {
		t.Fatalf("EnsureSession() error: %v", err)
	}
	runner.assertDone()
}
