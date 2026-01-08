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

func TestRunnerCommandsSmoke(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	var exitCode int = -1
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
	run := func(args ...string) error {
		out.Reset()
		deps := baseDeps
		deps.Stdout = &out
		deps.Stderr = &out
		deps.Stdin = strings.NewReader("")
		runner, err := NewRunner(deps)
		if err != nil {
			return err
		}
		exitCode = -1
		return runner.Run(context.Background(), args)
	}

	err := run("peky", "--version")
	if err != nil {
		if _, ok := err.(cli.ExitCoder); !ok {
			t.Fatalf("--version unexpected error: %T %v", err, err)
		}
	}
	if exitCode != 0 {
		t.Fatalf("--version exitCode=%d", exitCode)
	}
	if !strings.Contains(out.String(), "peky test") {
		t.Fatalf("--version output = %q", out.String())
	}

	err = run("peky", "version")
	if err != nil {
		t.Fatalf("version error: %v", err)
	}
	if !strings.Contains(out.String(), "peky test") {
		t.Fatalf("version output = %q", out.String())
	}

	err = run("peky", "layouts", "--json")
	if err != nil {
		t.Fatalf("layouts --json error: %v", err)
	}
	if !strings.Contains(out.String(), "\"ok\":true") {
		t.Fatalf("layouts --json output = %q", out.String())
	}

	err = run("peky", "--json", "layouts", "export", "auto")
	if err != nil {
		t.Fatalf("layouts export auto --json error: %v", err)
	}
	if !strings.Contains(out.String(), "\"name\":\"auto\"") {
		t.Fatalf("layouts export output = %q", out.String())
	}

	err = run("peky", "context", "pack", "--include", "errors", "--json")
	if err != nil {
		t.Fatalf("context pack error: %v", err)
	}
	if !strings.Contains(out.String(), "\"ok\":true") {
		t.Fatalf("context pack output = %q", out.String())
	}

	err = run("peky", "nl", "plan", "list", "sessions", "--json")
	if err != nil {
		t.Fatalf("nl plan error: %v", err)
	}
	if !strings.Contains(out.String(), "\"plan_id\"") {
		t.Fatalf("nl plan output = %q", out.String())
	}

	err = run("peky", "help")
	if err != nil {
		t.Fatalf("help error: %v", err)
	}
	if !strings.Contains(out.String(), "USAGE:") {
		t.Fatalf("help output = %q", out.String())
	}

	exitCode = -1
	err = run("peky", "start", "--json", "--yes")
	if err != nil {
		if _, ok := err.(cli.ExitCoder); !ok {
			t.Fatalf("start unexpected error: %T %v", err, err)
		}
	}
	if exitCode != 1 || !strings.Contains(out.String(), "daemon connection not configured") {
		t.Fatalf("start exitCode=%d out=%q", exitCode, out.String())
	}

	exitCode = -1
	err = run("peky", "events", "replay", "--json")
	if err != nil {
		if _, ok := err.(cli.ExitCoder); !ok {
			t.Fatalf("events replay unexpected error: %T %v", err, err)
		}
	}
	if exitCode != 1 || !strings.Contains(out.String(), "daemon connection not configured") {
		t.Fatalf("events replay exitCode=%d out=%q", exitCode, out.String())
	}

	exitCode = -1
	err = run("peky", "relay", "list", "--json")
	if err != nil {
		if _, ok := err.(cli.ExitCoder); !ok {
			t.Fatalf("relay list unexpected error: %T %v", err, err)
		}
	}
	if exitCode != 1 || !strings.Contains(out.String(), "daemon connection not configured") {
		t.Fatalf("relay list exitCode=%d out=%q", exitCode, out.String())
	}

	err = run("peky", "clone")
	if err == nil || !strings.Contains(err.Error(), "missing argument \"repo\"") {
		t.Fatalf("clone err=%v", err)
	}
}
