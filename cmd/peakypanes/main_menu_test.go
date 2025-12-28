package main

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/regenrek/peakypanes/internal/sessiond"
	"github.com/regenrek/peakypanes/internal/tui/app"
)

type fakeProgram struct {
	ran bool
	err error
}

func (p *fakeProgram) Run() (tea.Model, error) {
	p.ran = true
	return nil, p.err
}

func TestMainNoArgsRunsMenu(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	origRunMenu := runMenuFn
	defer func() { runMenuFn = origRunMenu }()

	called := false
	runMenuFn = func(_ *app.AutoStartSpec) { called = true }

	os.Args = []string{"peakypanes"}
	main()
	if !called {
		t.Fatalf("expected runMenu to be called")
	}
}

func TestMainVersionPrints(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	origVersion := version
	defer func() { version = origVersion }()

	version = "test-version"
	os.Args = []string{"peakypanes", "version"}
	out := captureStdout(func() {
		main()
	})
	if !strings.Contains(out, "test-version") {
		t.Fatalf("expected version output, got %q", out)
	}
}

func TestRunMenuWithSuccess(t *testing.T) {
	var (
		connectCalled   bool
		newModelCalled  bool
		openInputCalled bool
		cleanupCalled   bool
	)
	program := &fakeProgram{}
	err := runMenuWith(nil, menuDeps{
		connect: func(ctx context.Context, ver string) (*sessiond.Client, error) {
			connectCalled = true
			return &sessiond.Client{}, nil
		},
		newModel: func(*sessiond.Client) (*app.Model, error) {
			newModelCalled = true
			return &app.Model{}, nil
		},
		openInput: func() (*os.File, func(), error) {
			openInputCalled = true
			return os.Stdin, func() { cleanupCalled = true }, nil
		},
		newProgram: func(tea.Model, ...tea.ProgramOption) programRunner {
			return program
		},
	})
	if err != nil {
		t.Fatalf("runMenuWith() error = %v", err)
	}
	if !connectCalled || !newModelCalled || !openInputCalled {
		t.Fatalf("missing dependency calls: connect=%v newModel=%v openInput=%v", connectCalled, newModelCalled, openInputCalled)
	}
	if !cleanupCalled {
		t.Fatalf("expected cleanup to be called")
	}
	if !program.ran {
		t.Fatalf("expected program to run")
	}
}

func TestRunMenuWithErrors(t *testing.T) {
	err := runMenuWith(nil, menuDeps{
		connect: func(context.Context, string) (*sessiond.Client, error) {
			return nil, errors.New("no daemon")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "failed to connect") {
		t.Fatalf("expected connect error, got %v", err)
	}

	err = runMenuWith(nil, menuDeps{
		connect: func(context.Context, string) (*sessiond.Client, error) {
			return &sessiond.Client{}, nil
		},
		newModel: func(*sessiond.Client) (*app.Model, error) {
			return nil, errors.New("bad model")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "failed to initialize") {
		t.Fatalf("expected model error, got %v", err)
	}

	err = runMenuWith(nil, menuDeps{
		connect: func(context.Context, string) (*sessiond.Client, error) {
			return &sessiond.Client{}, nil
		},
		newModel: func(*sessiond.Client) (*app.Model, error) {
			return &app.Model{}, nil
		},
		openInput: func() (*os.File, func(), error) {
			return nil, func() {}, errors.New("no input")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot initialize TUI input") {
		t.Fatalf("expected input error, got %v", err)
	}

	err = runMenuWith(nil, menuDeps{
		connect: func(context.Context, string) (*sessiond.Client, error) {
			return &sessiond.Client{}, nil
		},
		newModel: func(*sessiond.Client) (*app.Model, error) {
			return &app.Model{}, nil
		},
		openInput: func() (*os.File, func(), error) {
			return os.Stdin, func() {}, nil
		},
		newProgram: func(tea.Model, ...tea.ProgramOption) programRunner {
			return &fakeProgram{err: errors.New("boom")}
		},
	})
	if err == nil || !strings.Contains(err.Error(), "TUI error") {
		t.Fatalf("expected program error, got %v", err)
	}
}

func TestRunMenuUsesDeps(t *testing.T) {
	origConnect := connectDefaultFn
	origNewModel := newModelFn
	origOpenInput := openTUIInputFn
	origNewProgram := newProgramFn
	origExit := exitFn
	defer func() {
		connectDefaultFn = origConnect
		newModelFn = origNewModel
		openTUIInputFn = origOpenInput
		newProgramFn = origNewProgram
		exitFn = origExit
	}()

	exitCalled := false
	exitFn = func(int) { exitCalled = true }

	connectDefaultFn = func(context.Context, string) (*sessiond.Client, error) {
		return &sessiond.Client{}, nil
	}
	newModelFn = func(*sessiond.Client) (*app.Model, error) {
		return &app.Model{}, nil
	}
	openTUIInputFn = func() (*os.File, func(), error) {
		return os.Stdin, func() {}, nil
	}
	program := &fakeProgram{}
	newProgramFn = func(tea.Model, ...tea.ProgramOption) programRunner {
		return program
	}

	runMenu(nil)
	if exitCalled {
		t.Fatalf("unexpected exit call")
	}
	if !program.ran {
		t.Fatalf("expected program run")
	}
}

func TestOpenTUIInput(t *testing.T) {
	origOpenFile := openFile
	origStdin := stdinFile
	defer func() {
		openFile = origOpenFile
		stdinFile = origStdin
	}()

	tmp, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = tmp.Close() }()

	openFile = func(string, int, os.FileMode) (*os.File, error) {
		return tmp, nil
	}
	stdinFile = tmp

	input, cleanup, err := openTUIInput()
	if err != nil {
		t.Fatalf("openTUIInput: %v", err)
	}
	if input != tmp {
		t.Fatalf("expected temp file input")
	}
	cleanup()
}

func TestOpenTUIInputWithTTY(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = tmp.Close() }()

	called := false
	input, cleanup, err := openTUIInputWith(
		func(string, int, os.FileMode) (*os.File, error) { return tmp, nil },
		func(*os.File) error { called = true; return nil },
		os.Stdin,
	)
	if err != nil {
		t.Fatalf("openTUIInputWith: %v", err)
	}
	if input != tmp {
		t.Fatalf("expected temp file input")
	}
	if !called {
		t.Fatalf("expected ensureBlocking to be called")
	}
	cleanup()
	if _, err := tmp.Stat(); err == nil {
		t.Fatalf("expected file to be closed")
	}
}

func TestOpenTUIInputWithStdinFallback(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = tmp.Close() }()

	called := false
	input, cleanup, err := openTUIInputWith(
		func(string, int, os.FileMode) (*os.File, error) { return nil, errors.New("no tty") },
		func(*os.File) error { called = true; return nil },
		tmp,
	)
	if err != nil {
		t.Fatalf("openTUIInputWith: %v", err)
	}
	if input != tmp {
		t.Fatalf("expected stdin fallback")
	}
	if !called {
		t.Fatalf("expected ensureBlocking to be called")
	}
	cleanup()
}

func TestOpenTUIInputWithStdinError(t *testing.T) {
	_, _, err := openTUIInputWith(
		func(string, int, os.FileMode) (*os.File, error) { return nil, errors.New("no tty") },
		func(*os.File) error { return errors.New("blocked") },
		os.Stdin,
	)
	if err == nil || !strings.Contains(err.Error(), "stdin is not a usable TUI input") {
		t.Fatalf("expected stdin error, got %v", err)
	}
}

func TestFatalWritesAndExits(t *testing.T) {
	oldExit := exitFn
	oldStderr := stderr
	defer func() {
		exitFn = oldExit
		stderr = oldStderr
	}()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	stderr = w

	code := 0
	exitFn = func(c int) { code = c }

	fatal("boom %s", "now")
	_ = w.Close()

	data, _ := io.ReadAll(r)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(string(data), "peakypanes: boom now") {
		t.Fatalf("unexpected stderr output: %q", string(data))
	}
}
