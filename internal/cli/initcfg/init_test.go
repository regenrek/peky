package initcfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/regenrek/peakypanes/internal/layout"
)

func TestInitLocalWritesConfig(t *testing.T) {
	dir := t.TempDir()
	if err := initLocal("peky", "auto", true, dir); err != nil {
		t.Fatalf("initLocal error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".peky.yml")); err != nil {
		t.Fatalf("expected .peky.yml: %v", err)
	}
}

func TestInitGlobalWritesConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	if err := initGlobal("peky", "auto", true); err != nil {
		t.Fatalf("initGlobal error: %v", err)
	}
	cfgPath, err := layout.DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath error: %v", err)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config: %v", err)
	}
}
