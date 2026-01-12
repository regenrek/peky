//go:build !windows

package sessiond

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/runenv"
)

func TestDefaultPathsEnvOverride(t *testing.T) {
	wantSocket := filepath.Join(t.TempDir(), "sock")
	wantPid := filepath.Join(t.TempDir(), "pid")

	t.Setenv(socketEnv, wantSocket)
	t.Setenv(pidEnv, wantPid)

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
}

func TestDefaultPathsRuntimeDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv(socketEnv, "")
	t.Setenv(pidEnv, "")
	t.Setenv(runenv.RuntimeDirEnv, "")

	socketPath, err := DefaultSocketPath()
	if err != nil {
		t.Fatalf("DefaultSocketPath: %v", err)
	}
	pidPath, err := DefaultPidPath()
	if err != nil {
		t.Fatalf("DefaultPidPath: %v", err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	runtimeDir := filepath.Join(configDir, identity.AppSlug)
	if _, err := os.Stat(runtimeDir); err != nil {
		t.Fatalf("expected runtime dir created: %v", err)
	}

	if socketPath != filepath.Join(runtimeDir, "daemon.sock") {
		t.Fatalf("socketPath = %q", socketPath)
	}
	if pidPath != filepath.Join(runtimeDir, "daemon.pid") {
		t.Fatalf("pidPath = %q", pidPath)
	}
}

func TestDefaultPathsRuntimeDirOverride(t *testing.T) {
	runtimeDir := t.TempDir()
	t.Setenv(runenv.RuntimeDirEnv, runtimeDir)
	t.Setenv(socketEnv, "")
	t.Setenv(pidEnv, "")

	socketPath, err := DefaultSocketPath()
	if err != nil {
		t.Fatalf("DefaultSocketPath: %v", err)
	}
	pidPath, err := DefaultPidPath()
	if err != nil {
		t.Fatalf("DefaultPidPath: %v", err)
	}

	if socketPath != filepath.Join(runtimeDir, "daemon.sock") {
		t.Fatalf("socketPath = %q", socketPath)
	}
	if pidPath != filepath.Join(runtimeDir, "daemon.pid") {
		t.Fatalf("pidPath = %q", pidPath)
	}
}
