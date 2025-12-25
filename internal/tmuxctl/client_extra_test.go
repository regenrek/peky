package tmuxctl

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestSourceFile(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"source-file", "/tmp/tmux.conf"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.SourceFile(context.Background(), "/tmp/tmux.conf"); err != nil {
		t.Fatalf("SourceFile() error: %v", err)
	}
	runner.assertDone()
}

func TestSourceFileValidation(t *testing.T) {
	client := &Client{bin: "tmux", run: func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCmd(ctx, "", "", 0)
	}}
	if err := client.SourceFile(context.Background(), ""); err == nil {
		t.Fatalf("SourceFile() expected error for empty path")
	}
}

func TestSelectWindow(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"select-window", "-t", "demo:1"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.SelectWindow(context.Background(), "demo:1"); err != nil {
		t.Fatalf("SelectWindow() error: %v", err)
	}
	runner.assertDone()
}

func TestKillSession(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"kill-session", "-t", "demo"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.KillSession(context.Background(), "demo"); err != nil {
		t.Fatalf("KillSession() error: %v", err)
	}
	runner.assertDone()
}

func TestRenameSession(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"rename-session", "-t", "old", "new"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.RenameSession(context.Background(), "old", "new"); err != nil {
		t.Fatalf("RenameSession() error: %v", err)
	}
	runner.assertDone()
}

func TestAttachSessionAndSwitchClient(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{name: "tmux", args: []string{"attach-session", "-t", "demo"}, exit: 0},
		{name: "tmux", args: []string{"switch-client", "-t", "demo"}, exit: 0},
	}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.AttachSession(context.Background(), "demo"); err != nil {
		t.Fatalf("AttachSession() error: %v", err)
	}
	if err := client.SwitchClient(context.Background(), "demo"); err != nil {
		t.Fatalf("SwitchClient() error: %v", err)
	}
	runner.assertDone()
}

func TestIsNestedTmuxErr(t *testing.T) {
	if isNestedTmuxErr(nil) {
		t.Fatalf("isNestedTmuxErr(nil) should be false")
	}
	if !isNestedTmuxErr(errTest("nested session detected")) {
		t.Fatalf("isNestedTmuxErr() should detect nested")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestSendKeysSlowImmediate(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{{
		name: "tmux",
		args: []string{"send-keys", "-t", "%1", "-l", "hi"},
		exit: 0,
	}}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.SendKeysSlow(context.Background(), "%1", "hi", 0); err != nil {
		t.Fatalf("SendKeysSlow() error: %v", err)
	}
	runner.assertDone()
}

func TestSendKeysSlowWithDelay(t *testing.T) {
	runner := &fakeRunner{t: t, specs: []cmdSpec{
		{name: "tmux", args: []string{"send-keys", "-t", "%1", "-l", "a"}, exit: 0},
		{name: "tmux", args: []string{"send-keys", "-t", "%1", "-l", "b"}, exit: 0},
	}}
	client := &Client{bin: "tmux", run: runner.run}
	if err := client.SendKeysSlow(context.Background(), "%1", "ab", time.Nanosecond); err != nil {
		t.Fatalf("SendKeysSlow() error: %v", err)
	}
	runner.assertDone()
}

func TestPasteTextBracketedSuccess(t *testing.T) {
	var calls [][]string
	run := func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{}, args...))
		switch args[0] {
		case "load-buffer":
			return helperCmd(ctx, "", "", 0)
		case "paste-buffer":
			return helperCmd(ctx, "", "", 0)
		default:
			t.Fatalf("unexpected command: %s %v", name, args)
			return helperCmd(ctx, "", "", 1)
		}
	}
	client := &Client{bin: "tmux", run: run}
	if err := client.PasteText(context.Background(), "%1", "hello", true); err != nil {
		t.Fatalf("PasteText() error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("PasteText() calls = %d", len(calls))
	}
	pasteArgs := calls[1]
	if !containsArg(pasteArgs, "-p") {
		t.Fatalf("PasteText() missing -p in %#v", pasteArgs)
	}
}

func TestSendBracketedPasteFallback(t *testing.T) {
	var calls [][]string
	run := func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls = append(calls, append([]string{}, args...))
		switch args[0] {
		case "load-buffer":
			return helperCmd(ctx, "nope", "", 1)
		case "send-keys":
			return helperCmd(ctx, "", "", 0)
		default:
			t.Fatalf("unexpected command: %s %v", name, args)
			return helperCmd(ctx, "", "", 1)
		}
	}
	client := &Client{bin: "tmux", run: run}
	if err := client.SendBracketedPaste(context.Background(), "%1", "hi"); err != nil {
		t.Fatalf("SendBracketedPaste() error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("SendBracketedPaste() calls = %d", len(calls))
	}
	payload := calls[1][4]
	if !strings.HasPrefix(payload, "\x1b[200~") || !strings.HasSuffix(payload, "\x1b[201~") {
		t.Fatalf("SendBracketedPaste() payload = %q", payload)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
