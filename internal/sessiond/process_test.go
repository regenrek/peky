package sessiond

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureDaemonRunningProbeShortCircuit(t *testing.T) {
	startCalled := false
	waitCalled := false
	err := ensureDaemonRunning(context.Background(), "v1", daemonOps{
		defaultSocketPath: func() (string, error) { return "sock", nil },
		probe:             func(context.Context, string, string) error { return nil },
		start:             func(string) error { startCalled = true; return nil },
		wait:              func(context.Context, string, string) error { waitCalled = true; return nil },
	})
	if err != nil {
		t.Fatalf("ensureDaemonRunning: %v", err)
	}
	if startCalled || waitCalled {
		t.Fatalf("unexpected start/wait: start=%v wait=%v", startCalled, waitCalled)
	}
}

func TestEnsureDaemonRunningStartAndWait(t *testing.T) {
	startCalled := false
	waitCalled := false
	err := ensureDaemonRunning(context.Background(), "v1", daemonOps{
		defaultSocketPath: func() (string, error) { return "sock", nil },
		probe:             func(context.Context, string, string) error { return errors.New("no daemon") },
		start:             func(string) error { startCalled = true; return nil },
		wait:              func(context.Context, string, string) error { waitCalled = true; return nil },
	})
	if err != nil {
		t.Fatalf("ensureDaemonRunning: %v", err)
	}
	if !startCalled || !waitCalled {
		t.Fatalf("expected start/wait to run: start=%v wait=%v", startCalled, waitCalled)
	}
}

func TestEnsureDaemonRunningProbeTimeout(t *testing.T) {
	startCalled := false
	err := ensureDaemonRunning(context.Background(), "v1", daemonOps{
		defaultSocketPath: func() (string, error) { return "sock", nil },
		probe:             func(context.Context, string, string) error { return ErrDaemonProbeTimeout },
		start:             func(string) error { startCalled = true; return nil },
		wait:              func(context.Context, string, string) error { return nil },
	})
	if !errors.Is(err, ErrDaemonProbeTimeout) {
		t.Fatalf("expected ErrDaemonProbeTimeout, got %v", err)
	}
	if startCalled {
		t.Fatalf("expected start to be skipped on probe timeout")
	}
}

func TestEnsureDaemonRunningStartError(t *testing.T) {
	want := errors.New("start failed")
	err := ensureDaemonRunning(context.Background(), "v1", daemonOps{
		defaultSocketPath: func() (string, error) { return "sock", nil },
		probe:             func(context.Context, string, string) error { return errors.New("no daemon") },
		start:             func(string) error { return want },
		wait:              func(context.Context, string, string) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), want.Error()) {
		t.Fatalf("expected start error, got %v", err)
	}
}

func TestConnectDefaultUsesDeps(t *testing.T) {
	ensureCalled := false
	dialCalled := false
	_, err := connectDefault(context.Background(), "v2", connectOps{
		defaultSocketPath: func() (string, error) { return "sock", nil },
		ensureRunning: func(ctx context.Context, version string) error {
			ensureCalled = true
			if version != "v2" {
				t.Fatalf("unexpected version %q", version)
			}
			return nil
		},
		dial: func(ctx context.Context, socketPath, version string) (*Client, error) {
			dialCalled = true
			if socketPath != "sock" || version != "v2" {
				t.Fatalf("unexpected dial args: %q %q", socketPath, version)
			}
			return &Client{}, nil
		},
	})
	if err != nil {
		t.Fatalf("connectDefault: %v", err)
	}
	if !ensureCalled || !dialCalled {
		t.Fatalf("expected ensure/dial: ensure=%v dial=%v", ensureCalled, dialCalled)
	}
}

func TestStartDaemonProcessWithDeps(t *testing.T) {
	socket := "sock-path"
	logPath := filepath.Join(t.TempDir(), "sessiond.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer func() { _ = logFile.Close() }()

	var gotExe string
	var gotArgs []string
	startCalled := false
	releaseCalled := false

	err = startDaemonProcessWith(socket, daemonProcessDeps{
		executable: func() (string, error) { return "/bin/true", nil },
		execCommand: func(name string, args ...string) *exec.Cmd {
			gotExe = name
			gotArgs = append([]string{}, args...)
			return &exec.Cmd{Path: name, Args: append([]string{name}, args...)}
		},
		defaultLog: func() (string, error) { return logPath, nil },
		environ:    func() []string { return []string{"TEST=1"} },
		openFile: func(string, int, os.FileMode) (*os.File, error) {
			return logFile, nil
		},
		startProcess: func(cmd *exec.Cmd) error {
			startCalled = true
			if cmd.Stdout == nil || cmd.Stderr == nil {
				t.Fatalf("expected stdout/stderr to be set")
			}
			if cmd.SysProcAttr == nil {
				t.Fatalf("expected SysProcAttr to be set")
			}
			found := false
			for _, kv := range cmd.Env {
				if kv == socketEnv+"="+socket {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected socket env to be set")
			}
			cmd.Process = &os.Process{Pid: os.Getpid()}
			return nil
		},
		releaseProc: func(*os.Process) error {
			releaseCalled = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("startDaemonProcessWith: %v", err)
	}
	if gotExe == "" || len(gotArgs) == 0 || gotArgs[0] != "daemon" {
		t.Fatalf("unexpected exec args: exe=%q args=%v", gotExe, gotArgs)
	}
	if !startCalled || !releaseCalled {
		t.Fatalf("expected start/release: start=%v release=%v", startCalled, releaseCalled)
	}
}

func TestStartDaemonProcessExecutableError(t *testing.T) {
	err := startDaemonProcessWith("sock", daemonProcessDeps{
		executable: func() (string, error) { return "", errors.New("missing") },
	})
	if err == nil || !strings.Contains(err.Error(), "resolve executable") {
		t.Fatalf("expected executable error, got %v", err)
	}
}

func TestHandleSignalsNil(t *testing.T) {
	var d *Daemon
	d.handleSignals()
}

func TestWaitForDaemonCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForDaemon(ctx, "/tmp/missing.sock", "v1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRunStop(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "sessiond.sock")
	pidPath := filepath.Join(dir, "sessiond.pid")

	d, err := NewDaemon(DaemonConfig{SocketPath: socketPath, PidPath: pidPath, Version: "test"})
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	d.manager = nil

	done := make(chan error, 1)
	go func() { done <- d.Run() }()

	waitForListener(t, d)
	if err := d.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func waitForListener(t *testing.T, d *Daemon) {
	t.Helper()
	for i := 0; i < 20000; i++ {
		if d.listenerValue() != nil {
			return
		}
		runtime.Gosched()
	}
	t.Fatalf("listener not set")
}
