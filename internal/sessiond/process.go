package sessiond

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type daemonOps struct {
	defaultSocketPath func() (string, error)
	probe             func(context.Context, string, string) error
	start             func(string) error
	wait              func(context.Context, string, string) error
}

type connectOps struct {
	defaultSocketPath func() (string, error)
	ensureRunning     func(context.Context, string) error
	dial              func(context.Context, string, string) (*Client, error)
}

type daemonProcessDeps struct {
	executable   func() (string, error)
	execCommand  func(string, ...string) *exec.Cmd
	defaultLog   func() (string, error)
	environ      func() []string
	openFile     func(string, int, os.FileMode) (*os.File, error)
	releaseProc  func(*os.Process) error
	startProcess func(*exec.Cmd) error
}

// EnsureDaemonRunning starts the daemon if needed.
func EnsureDaemonRunning(ctx context.Context, version string) error {
	return ensureDaemonRunning(ctx, version, daemonOps{
		defaultSocketPath: DefaultSocketPath,
		probe:             probeDaemon,
		start:             startDaemonProcess,
		wait:              waitForDaemon,
	})
}

// ConnectDefault ensures the daemon is running and returns a client.
func ConnectDefault(ctx context.Context, version string) (*Client, error) {
	return connectDefault(ctx, version, connectOps{
		defaultSocketPath: DefaultSocketPath,
		ensureRunning:     EnsureDaemonRunning,
		dial:              Dial,
	})
}

// StopDaemon attempts to stop a running daemon instance.
func StopDaemon(ctx context.Context, version string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	socketPath, err := DefaultSocketPath()
	if err != nil {
		return err
	}
	pidPath, err := DefaultPidPath()
	if err != nil {
		return err
	}
	if err := probeDaemon(ctx, socketPath, version); err != nil {
		if errors.Is(err, ErrDaemonProbeTimeout) {
			return err
		}
		return nil
	}
	pid, err := readPidFile(pidPath)
	if err != nil {
		return err
	}
	if err := signalDaemon(pid); err != nil {
		return err
	}
	return waitForDaemonStop(ctx, socketPath, version)
}

// RestartDaemon stops the daemon (if running) and starts a new instance.
func RestartDaemon(ctx context.Context, version string) error {
	if err := StopDaemon(ctx, version); err != nil {
		return err
	}
	socketPath, err := DefaultSocketPath()
	if err != nil {
		return err
	}
	if err := startDaemonProcess(socketPath); err != nil {
		return err
	}
	return waitForDaemon(ctx, socketPath, version)
}

func ensureDaemonRunning(ctx context.Context, version string, ops daemonOps) error {
	defaultSocketPath := ops.defaultSocketPath
	if defaultSocketPath == nil {
		defaultSocketPath = DefaultSocketPath
	}
	probe := ops.probe
	if probe == nil {
		probe = probeDaemon
	}
	start := ops.start
	if start == nil {
		start = startDaemonProcess
	}
	wait := ops.wait
	if wait == nil {
		wait = waitForDaemon
	}

	socketPath, err := defaultSocketPath()
	if err != nil {
		return err
	}
	if err := probe(ctx, socketPath, version); err == nil {
		return nil
	} else if errors.Is(err, ErrDaemonProbeTimeout) {
		return err
	}
	if err := start(socketPath); err != nil {
		return err
	}
	return wait(ctx, socketPath, version)
}

func connectDefault(ctx context.Context, version string, ops connectOps) (*Client, error) {
	defaultSocketPath := ops.defaultSocketPath
	if defaultSocketPath == nil {
		defaultSocketPath = DefaultSocketPath
	}
	ensureRunning := ops.ensureRunning
	if ensureRunning == nil {
		ensureRunning = EnsureDaemonRunning
	}
	dial := ops.dial
	if dial == nil {
		dial = Dial
	}

	socketPath, err := defaultSocketPath()
	if err != nil {
		return nil, err
	}
	if err := ensureRunning(ctx, version); err != nil {
		return nil, err
	}
	return dial(ctx, socketPath, version)
}

func startDaemonProcess(socketPath string) error {
	return startDaemonProcessWith(socketPath, daemonProcessDeps{
		executable:   os.Executable,
		execCommand:  exec.Command,
		defaultLog:   DefaultLogPath,
		environ:      os.Environ,
		openFile:     os.OpenFile,
		releaseProc:  func(p *os.Process) error { return p.Release() },
		startProcess: func(cmd *exec.Cmd) error { return cmd.Start() },
	})
}

func startDaemonProcessWith(socketPath string, deps daemonProcessDeps) error {
	executable := deps.executable
	if executable == nil {
		executable = os.Executable
	}
	execCommand := deps.execCommand
	if execCommand == nil {
		execCommand = exec.Command
	}
	defaultLog := deps.defaultLog
	if defaultLog == nil {
		defaultLog = DefaultLogPath
	}
	environ := deps.environ
	if environ == nil {
		environ = os.Environ
	}
	openFile := deps.openFile
	if openFile == nil {
		openFile = os.OpenFile
	}
	releaseProc := deps.releaseProc
	if releaseProc == nil {
		releaseProc = func(p *os.Process) error { return p.Release() }
	}
	startProcess := deps.startProcess
	if startProcess == nil {
		startProcess = func(cmd *exec.Cmd) error { return cmd.Start() }
	}

	exe, err := executable()
	if err != nil {
		return fmt.Errorf("sessiond: resolve executable: %w", err)
	}
	cmd := execCommand(exe, "daemon")
	configureDaemonCommand(cmd)
	cmd.Env = append(environ(), socketEnv+"="+socketPath)

	logPath, err := defaultLog()
	if err == nil && logPath != "" {
		if file, openErr := openFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); openErr == nil {
			cmd.Stdout = file
			cmd.Stderr = file
		}
	}

	if err := startProcess(cmd); err != nil {
		return fmt.Errorf("sessiond: start daemon: %w", err)
	}
	if cmd.Process != nil {
		_ = releaseProc(cmd.Process)
	}
	return nil
}

func waitForDaemon(ctx context.Context, socketPath, version string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("sessiond: daemon did not start")
		case <-ticker.C:
			if err := probeDaemon(ctx, socketPath, version); err == nil {
				return nil
			}
		}
	}
}

func waitForDaemonStop(ctx context.Context, socketPath, version string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := os.Stat(socketPath); err != nil {
				if os.IsNotExist(err) {
					return nil
				}
			}
			if err := probeDaemon(ctx, socketPath, version); err != nil {
				if errors.Is(err, ErrDaemonProbeTimeout) {
					return err
				}
				return nil
			}
		}
	}
}

func readPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("sessiond: read pid file: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return 0, fmt.Errorf("sessiond: pid file empty")
	}
	pid, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("sessiond: parse pid: %w", err)
	}
	return pid, nil
}
