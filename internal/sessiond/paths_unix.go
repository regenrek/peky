//go:build !windows

package sessiond

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/regenrek/peakypanes/internal/appdirs"
)

// DefaultSocketPath returns the default unix socket path.
func DefaultSocketPath() (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return ResolveSocketPath(dir), nil
}

// DefaultPidPath returns the default pid file path.
func DefaultPidPath() (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return ResolvePidPath(dir), nil
}

// ResolveSocketPath returns the effective socket path for the provided runtime dir.
func ResolveSocketPath(runtimeDir string) string {
	if path := strings.TrimSpace(os.Getenv(socketEnv)); path != "" {
		return path
	}
	runtimeDir = strings.TrimSpace(runtimeDir)
	if runtimeDir == "" {
		return ""
	}
	return filepath.Join(runtimeDir, "daemon.sock")
}

// ResolvePidPath returns the effective pid path for the provided runtime dir.
func ResolvePidPath(runtimeDir string) string {
	if path := strings.TrimSpace(os.Getenv(pidEnv)); path != "" {
		return path
	}
	runtimeDir = strings.TrimSpace(runtimeDir)
	if runtimeDir == "" {
		return ""
	}
	return filepath.Join(runtimeDir, "daemon.pid")
}

func runtimeDir() (string, error) {
	return appdirs.RuntimeDir()
}
