//go:build !windows

package sessiond

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/regenrek/peakypanes/internal/runenv"
)

const (
	socketEnv = "PEAKYPANES_DAEMON_SOCKET"
	pidEnv    = "PEAKYPANES_DAEMON_PID"
	logEnv    = "PEAKYPANES_DAEMON_LOG"
)

// DefaultSocketPath returns the default unix socket path.
func DefaultSocketPath() (string, error) {
	if path := os.Getenv(socketEnv); path != "" {
		return path, nil
	}
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.sock"), nil
}

// DefaultPidPath returns the default pid file path.
func DefaultPidPath() (string, error) {
	if path := os.Getenv(pidEnv); path != "" {
		return path, nil
	}
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.pid"), nil
}

// DefaultLogPath returns the default log file path.
func DefaultLogPath() (string, error) {
	if path := os.Getenv(logEnv); path != "" {
		return path, nil
	}
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.log"), nil
}

func runtimeDir() (string, error) {
	if override := runenv.RuntimeDir(); override != "" {
		if err := os.MkdirAll(override, 0o755); err != nil {
			return "", fmt.Errorf("create runtime dir: %w", err)
		}
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	dir = filepath.Join(dir, "peakypanes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create runtime dir: %w", err)
	}
	return dir, nil
}
