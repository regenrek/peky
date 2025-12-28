//go:build windows

package sessiond

import "errors"

const (
	socketEnv = "PEAKYPANES_DAEMON_SOCKET"
	pidEnv    = "PEAKYPANES_DAEMON_PID"
	logEnv    = "PEAKYPANES_DAEMON_LOG"
)

// DefaultSocketPath returns the default socket path on Windows.
func DefaultSocketPath() (string, error) {
	return "", errors.New("session daemon sockets are not supported on windows yet")
}

// DefaultPidPath returns the default pid file path on Windows.
func DefaultPidPath() (string, error) {
	return "", errors.New("session daemon pid files are not supported on windows yet")
}

// DefaultLogPath returns the default log file path on Windows.
func DefaultLogPath() (string, error) {
	return "", errors.New("session daemon logs are not supported on windows yet")
}
