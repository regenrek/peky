package atomicfile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveWritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	data := []byte("hello")

	if err := Save(path, data, 0o600); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("content = %q, want %q", string(got), string(data))
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat() error: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("perm = %o, want 0600", info.Mode().Perm())
		}
	}
}

func TestSaveEmptyPath(t *testing.T) {
	if err := Save("", []byte("x"), 0o600); err == nil {
		t.Fatalf("expected error for empty path")
	}
}
