//go:build !windows

package appdirs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeDirPermissions(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "runtime")
	t.Setenv("PEKY_RUNTIME_DIR", dir)

	got, err := RuntimeDir()
	if err != nil {
		t.Fatalf("RuntimeDir() error: %v", err)
	}
	if got != dir {
		t.Fatalf("RuntimeDir() = %q, want %q", got, dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat runtime dir: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("runtime dir perm = %o, want 0700", info.Mode().Perm())
	}
}

func TestEnsureRuntimeDirTightensDefaultPerms(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "runtime")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod runtime dir: %v", err)
	}

	got, err := ensureRuntimeDir(dir, false)
	if err != nil {
		t.Fatalf("ensureRuntimeDir() error: %v", err)
	}
	if got != dir {
		t.Fatalf("ensureRuntimeDir() = %q, want %q", got, dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat runtime dir: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("runtime dir perm = %o, want 0700", info.Mode().Perm())
	}
}

func TestDataDirPermissions(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "data")
	t.Setenv("PEKY_DATA_DIR", dir)

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() error: %v", err)
	}
	if got != dir {
		t.Fatalf("DataDir() = %q, want %q", got, dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("data dir perm = %o, want 0700", info.Mode().Perm())
	}
}

func TestEnsureDataDirTightensDefaultPerms(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "data")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod data dir: %v", err)
	}

	got, err := ensureDataDir(dir, false)
	if err != nil {
		t.Fatalf("ensureDataDir() error: %v", err)
	}
	if got != dir {
		t.Fatalf("ensureDataDir() = %q, want %q", got, dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("data dir perm = %o, want 0700", info.Mode().Perm())
	}
}
