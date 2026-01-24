package sessiond

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/logging"
)

func TestDefaultDaemonLogFileEnv(t *testing.T) {
	t.Setenv(logging.EnvLogFile, "/tmp/test-daemon.log")
	path, err := defaultDaemonLogFile()
	if err != nil {
		t.Fatalf("defaultDaemonLogFile error: %v", err)
	}
	if path != "/tmp/test-daemon.log" {
		t.Fatalf("path=%q", path)
	}
}

func TestReadPidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pid")
	if err := os.WriteFile(path, []byte("123"), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	if pid, err := readPidFile(path); err != nil || pid != 123 {
		t.Fatalf("pid=%d err=%v", pid, err)
	}
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	if _, err := readPidFile(path); err == nil {
		t.Fatalf("expected empty pid error")
	}
	if err := os.WriteFile(path, []byte("nope"), 0o644); err != nil {
		t.Fatalf("write pid: %v", err)
	}
	if _, err := readPidFile(path); err == nil {
		t.Fatalf("expected parse error")
	}
}
