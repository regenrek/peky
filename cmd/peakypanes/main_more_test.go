package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/app"
)

func TestMainHelpAndDaemonHelp(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"peakypanes", "-z"}
	out := captureStdout(func() { main() })
	if !strings.Contains(out, "Usage") {
		t.Fatalf("expected help output, got %q", out)
	}

	os.Args = []string{"peakypanes", "daemon", "--help"}
	out = captureStdout(func() { main() })
	if !strings.Contains(out, "session daemon") {
		t.Fatalf("expected daemon help output, got %q", out)
	}
}

func TestMainCloneBranch(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origRunMenu := runMenuFn
	defer func() { runMenuFn = origRunMenu }()

	var got *app.AutoStartSpec
	runMenuFn = func(spec *app.AutoStartSpec) { got = spec }

	home := t.TempDir()
	t.Setenv("HOME", home)
	target := filepath.Join(home, "projects", "repo")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	os.Args = []string{"peakypanes", "clone", "user/repo"}
	main()
	if got == nil || got.Path != target {
		t.Fatalf("expected autoStart path %q, got %#v", target, got)
	}
}

func TestRunMenuErrorPath(t *testing.T) {
	origConnect := connectDefaultFn
	origExit := exitFn
	origStderr := stderr
	defer func() {
		connectDefaultFn = origConnect
		exitFn = origExit
		stderr = origStderr
	}()

	connectDefaultFn = func(context.Context, string) (*sessiond.Client, error) {
		return nil, errors.New("boom")
	}
	code := 0
	exitFn = func(c int) { code = c }
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	stderr = w

	runMenu(nil)
	_ = w.Close()
	data, _ := io.ReadAll(r)
	if code == 0 {
		t.Fatalf("expected exit code")
	}
	if !strings.Contains(string(data), "failed to connect") {
		t.Fatalf("unexpected stderr: %q", string(data))
	}
}

func TestRunLayoutsExportMissing(t *testing.T) {
	oldExit := exitFn
	defer func() {
		exitFn = oldExit
	}()

	exitFn = func(int) { panic("exit") }
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic from fatal")
		}
	}()

	runLayouts([]string{"export"})
}

func TestOpenTUIInputWithConfigureError(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = tmp.Close() }()

	_, _, err = openTUIInputWith(
		func(string, int, os.FileMode) (*os.File, error) { return tmp, nil },
		func(*os.File) error { return errors.New("bad") },
		os.Stdin,
	)
	if err == nil || !strings.Contains(err.Error(), "configure /dev/tty") {
		t.Fatalf("expected configure error, got %v", err)
	}
}

func TestRunInitHelp(t *testing.T) {
	out := captureStdout(func() {
		runInit([]string{"--help"})
	})
	if !strings.Contains(out, "Initialize Peaky Panes configuration") {
		t.Fatalf("expected init help output, got %q", out)
	}
}
