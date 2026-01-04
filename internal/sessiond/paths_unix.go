//go:build !windows

package sessiond

import (
	"os"
	"path/filepath"

	"github.com/regenrek/peakypanes/internal/appdirs"
)

const (
	socketEnv = "PEAKYPANES_DAEMON_SOCKET"
	pidEnv    = "PEAKYPANES_DAEMON_PID"
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

func runtimeDir() (string, error) {
	return appdirs.RuntimeDir()
}
