package sessiond

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// EnsureDaemonRunning starts the daemon if needed.
func EnsureDaemonRunning(ctx context.Context, version string) error {
	socketPath, err := DefaultSocketPath()
	if err != nil {
		return err
	}
	if err := probeDaemon(ctx, socketPath, version); err == nil {
		return nil
	}
	if err := startDaemonProcess(socketPath); err != nil {
		return err
	}
	return waitForDaemon(ctx, socketPath, version)
}

// ConnectDefault ensures the daemon is running and returns a client.
func ConnectDefault(ctx context.Context, version string) (*Client, error) {
	socketPath, err := DefaultSocketPath()
	if err != nil {
		return nil, err
	}
	if err := EnsureDaemonRunning(ctx, version); err != nil {
		return nil, err
	}
	return Dial(ctx, socketPath, version)
}

func startDaemonProcess(socketPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("sessiond: resolve executable: %w", err)
	}
	cmd := exec.Command(exe, "daemon")
	cmd.Env = append(os.Environ(), socketEnv+"="+socketPath)

	logPath, err := DefaultLogPath()
	if err == nil && logPath != "" {
		if file, openErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); openErr == nil {
			cmd.Stdout = file
			cmd.Stderr = file
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("sessiond: start daemon: %w", err)
	}
	_ = cmd.Process.Release()
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
