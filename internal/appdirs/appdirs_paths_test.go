//go:build !windows

package appdirs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/runenv"
)

func TestRuntimeDirPathOverrideDoesNotCreate(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "runtime")
	t.Setenv(runenv.RuntimeDirEnv, dir)

	got, err := RuntimeDirPath()
	if err != nil {
		t.Fatalf("RuntimeDirPath() error: %v", err)
	}
	if got != dir {
		t.Fatalf("RuntimeDirPath() = %q, want %q", got, dir)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected runtime dir to not exist, err=%v", err)
	}
}

func TestDataDirPathOverrideDoesNotCreate(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "data")
	t.Setenv(runenv.DataDirEnv, dir)

	got, err := DataDirPath()
	if err != nil {
		t.Fatalf("DataDirPath() error: %v", err)
	}
	if got != dir {
		t.Fatalf("DataDirPath() = %q, want %q", got, dir)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected data dir to not exist, err=%v", err)
	}
}
