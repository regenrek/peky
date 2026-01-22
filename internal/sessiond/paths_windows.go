//go:build windows

package sessiond

import (
	"errors"
	"os"
	"strings"
)

// DefaultSocketPath returns the default socket path on Windows.
func DefaultSocketPath() (string, error) {
	return "", errors.New("session daemon sockets are not supported on windows yet")
}

// DefaultPidPath returns the default pid file path on Windows.
func DefaultPidPath() (string, error) {
	return "", errors.New("session daemon pid files are not supported on windows yet")
}

// ResolveSocketPath returns the effective socket path for the provided runtime dir.
func ResolveSocketPath(runtimeDir string) string {
	if path := strings.TrimSpace(os.Getenv(socketEnv)); path != "" {
		return path
	}
	return ""
}

// ResolvePidPath returns the effective pid path for the provided runtime dir.
func ResolvePidPath(runtimeDir string) string {
	if path := strings.TrimSpace(os.Getenv(pidEnv)); path != "" {
		return path
	}
	return ""
}
