package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProjectConfigStateForPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".peky.yml")
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

func TestProjectConfigStateEqual(t *testing.T) {
	base := projectConfigState{
		path:    "/tmp/app.yml",
		modTime: time.Unix(10, 0),
		size:    42,
		exists:  true,
	}
	same := projectConfigState{
		path:    "/tmp/app.yml",
		modTime: time.Unix(10, 0),
		size:    42,
		exists:  true,
	}
	if !base.equal(same) {
		t.Fatalf("expected states equal")
	}
	changed := base
	changed.size = 7
	if base.equal(changed) {
		t.Fatalf("expected states not equal")
	}
}
