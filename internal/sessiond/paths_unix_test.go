//go:build !windows

package sessiond

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPathsEnvOverride(t *testing.T) {
	wantSocket := filepath.Join(t.TempDir(), "sock")
	wantPid := filepath.Join(t.TempDir(), "pid")
	wantLog := filepath.Join(t.TempDir(), "log")

	t.Setenv(socketEnv, wantSocket)
	t.Setenv(pidEnv, wantPid)
	t.Setenv(logEnv, wantLog)

	if got, err := DefaultSocketPath(); err != nil {
		t.Fatalf("DefaultSocketPath: %v", err)
	} else if got != wantSocket {
		t.Fatalf("DefaultSocketPath = %q", got)
	}

	if got, err := DefaultPidPath(); err != nil {
		t.Fatalf("DefaultPidPath: %v", err)
	} else if got != wantPid {
		t.Fatalf("DefaultPidPath = %q", got)
	}

	if got, err := DefaultLogPath(); err != nil {
		t.Fatalf("DefaultLogPath: %v", err)
	} else if got != wantLog {
		t.Fatalf("DefaultLogPath = %q", got)
	}
}

func TestDefaultPathsRuntimeDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv(socketEnv, "")
	t.Setenv(pidEnv, "")
	t.Setenv(logEnv, "")

	socketPath, err := DefaultSocketPath()
	if err != nil {
		t.Fatalf("DefaultSocketPath: %v", err)
	}
	pidPath, err := DefaultPidPath()
	if err != nil {
		t.Fatalf("DefaultPidPath: %v", err)
	}
	logPath, err := DefaultLogPath()
	if err != nil {
		t.Fatalf("DefaultLogPath: %v", err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	runtimeDir := filepath.Join(configDir, "peakypanes")
	if _, err := os.Stat(runtimeDir); err != nil {
		t.Fatalf("expected runtime dir created: %v", err)
	}

	if socketPath != filepath.Join(runtimeDir, "daemon.sock") {
		t.Fatalf("socketPath = %q", socketPath)
	}
	if pidPath != filepath.Join(runtimeDir, "daemon.pid") {
		t.Fatalf("pidPath = %q", pidPath)
	}
	if logPath != filepath.Join(runtimeDir, "daemon.log") {
		t.Fatalf("logPath = %q", logPath)
	}
}
