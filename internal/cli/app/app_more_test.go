package app

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/regenrek/peakypanes/internal/cli/root"
)

type runnerHarness struct {
	run func(args ...string) (out string, exitCode int, err error)
}

func newRunnerHarness(t *testing.T) runnerHarness {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	exitCode := -1
	prevExiter := cli.OsExiter
	prevErrWriter := cli.ErrWriter
	cli.OsExiter = func(code int) { exitCode = code }
	cli.ErrWriter = io.Discard
	t.Cleanup(func() {
		cli.OsExiter = prevExiter
		cli.ErrWriter = prevErrWriter
	})

	var out bytes.Buffer
	baseDeps := root.Dependencies{
		Version: "test",
		AppName: "peky",
		Connect: nil,
	}
	run := func(args ...string) (string, int, error) {
		out.Reset()
		deps := baseDeps
		deps.Stdout = &out
		deps.Stderr = &out
		deps.Stdin = strings.NewReader("")
		runner, err := NewRunner(deps)
		if err != nil {
			return "", -1, err
		}
		exitCode = -1
		err = runner.Run(context.Background(), args)
		return out.String(), normalizeExitCode(exitCode, err), err
	}
	return runnerHarness{run: run}
}

func normalizeExitCode(exitCode int, err error) int {
	if err != nil {
		if ec, ok := err.(cli.ExitCoder); ok {
			return ec.ExitCode()
		}
	}
	if exitCode >= 0 {
		return exitCode
	}
	if err == nil {
		return 0
	}
	return -1
}

func requireExitCode(t *testing.T, got, want int, out string) {
	t.Helper()
	if got != want {
		t.Fatalf("exitCode=%d want=%d out=%q", got, want, out)
	}
}

func requireContains(t *testing.T, out, want string) {
	t.Helper()
	if !strings.Contains(out, want) {
		t.Fatalf("out=%q want substring %q", out, want)
	}
}

func requireExitCoderOrNil(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if _, ok := err.(cli.ExitCoder); ok {
		return
	}
	t.Fatalf("unexpected error: %T %v", err, err)
}

func TestRunnerVersionFlag(t *testing.T) {
	h := newRunnerHarness(t)

	out, code, err := h.run("peky", "--version")
	requireExitCoderOrNil(t, err)
	requireExitCode(t, code, 0, out)
	requireContains(t, out, "peky test")
}

func TestRunnerVersionCommand(t *testing.T) {
	h := newRunnerHarness(t)

	out, code, err := h.run("peky", "version")
	if err != nil {
		t.Fatalf("version error: %v", err)
	}
	requireExitCode(t, code, 0, out)
	requireContains(t, out, "peky test")
}

func TestRunnerJSONCommands(t *testing.T) {
	h := newRunnerHarness(t)

	out, code, err := h.run("peky", "layouts", "--json")
	if err != nil {
		t.Fatalf("layouts --json error: %v", err)
	}
	requireExitCode(t, code, 0, out)
	requireContains(t, out, "\"ok\":true")

	out, code, err = h.run("peky", "--json", "layouts", "export", "auto")
	if err != nil {
		t.Fatalf("layouts export auto --json error: %v", err)
	}
	requireExitCode(t, code, 0, out)
	requireContains(t, out, "\"name\":\"auto\"")

	out, code, err = h.run("peky", "context", "pack", "--include", "errors", "--json")
	if err != nil {
		t.Fatalf("context pack error: %v", err)
	}
	requireExitCode(t, code, 0, out)
	requireContains(t, out, "\"ok\":true")

	out, code, err = h.run("peky", "nl", "plan", "list", "sessions", "--json")
	if err != nil {
		t.Fatalf("nl plan error: %v", err)
	}
	requireExitCode(t, code, 0, out)
	requireContains(t, out, "\"plan_id\"")
}

func TestRunnerHelp(t *testing.T) {
	h := newRunnerHarness(t)

	out, code, err := h.run("peky", "help")
	if err != nil {
		t.Fatalf("help error: %v", err)
	}
	requireExitCode(t, code, 0, out)
	requireContains(t, out, "USAGE:")
}

func TestRunnerCommandsNeedDaemon(t *testing.T) {
	h := newRunnerHarness(t)

	cases := []struct {
		args       []string
		wantSubstr string
	}{
		{args: []string{"peky", "start", "--json", "--yes"}, wantSubstr: "daemon connection not configured"},
		{args: []string{"peky", "events", "replay", "--json"}, wantSubstr: "daemon connection not configured"},
		{args: []string{"peky", "relay", "list", "--json"}, wantSubstr: "daemon connection not configured"},
	}
	for _, c := range cases {
		out, code, err := h.run(c.args...)
		requireExitCoderOrNil(t, err)
		requireExitCode(t, code, 1, out)
		requireContains(t, out, c.wantSubstr)
	}
}

func TestRunnerCloneNeedsRepo(t *testing.T) {
	h := newRunnerHarness(t)

	out, code, err := h.run("peky", "clone")
	if err == nil || !strings.Contains(err.Error(), "missing argument \"repo\"") {
		t.Fatalf("err=%v out=%q code=%d", err, out, code)
	}
}
