//go:build !windows

package logging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLogDirTightensPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := ensureLogDir(dir, false); err != nil {
		t.Fatalf("ensureLogDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected 0700 permissions, got %v", info.Mode().Perm())
	}
}

func TestEnsureLogDirOverrideWarnsOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "override")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := ensureLogDir(dir, true); err != nil {
		t.Fatalf("ensureLogDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0o077 == 0 {
		t.Fatalf("expected override dir permissions unchanged, got %v", info.Mode().Perm())
	}
}
