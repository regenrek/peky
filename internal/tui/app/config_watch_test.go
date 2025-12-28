package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectConfigStateForPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".peakypanes.yml")
	if err := os.WriteFile(path, []byte("ok"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	state := projectConfigStateForPath(dir)
	if !state.exists || state.path != path {
		t.Fatalf("unexpected state: %#v", state)
	}

	if state := projectConfigStateForPath(filepath.Join(dir, "missing")); state.exists {
		t.Fatalf("expected missing state")
	}
}
