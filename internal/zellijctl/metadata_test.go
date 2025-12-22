package zellijctl

import (
	"path/filepath"
	"testing"
)

func TestRecordAndLoadSessionPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg"))

	if err := RecordSessionPath("session-a", "/projects/a"); err != nil {
		t.Fatalf("RecordSessionPath: %v", err)
	}
	if err := RecordSessionPath("session-b", "/projects/b"); err != nil {
		t.Fatalf("RecordSessionPath: %v", err)
	}

	paths, err := LoadSessionPaths()
	if err != nil {
		t.Fatalf("LoadSessionPaths: %v", err)
	}
	if got := paths["session-a"]; got != "/projects/a" {
		t.Fatalf("session-a path mismatch: %q", got)
	}
	if got := paths["session-b"]; got != "/projects/b" {
		t.Fatalf("session-b path mismatch: %q", got)
	}

	if err := RecordSessionPath("session-a", "/projects/a2"); err != nil {
		t.Fatalf("RecordSessionPath update: %v", err)
	}
	paths, err = LoadSessionPaths()
	if err != nil {
		t.Fatalf("LoadSessionPaths second: %v", err)
	}
	if got := paths["session-a"]; got != "/projects/a2" {
		t.Fatalf("session-a updated path mismatch: %q", got)
	}
}
