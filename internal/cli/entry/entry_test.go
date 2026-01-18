package entry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regenrek/peakypanes/internal/identity"
	"github.com/regenrek/peakypanes/internal/runenv"
)

func TestRunCreatesDefaultConfig(t *testing.T) {
	configDir := t.TempDir()
	runtimeDir := t.TempDir()
	t.Setenv(runenv.ConfigDirEnv, configDir)
	t.Setenv(runenv.RuntimeDirEnv, runtimeDir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	configPath := filepath.Join(configDir, identity.GlobalConfigFile)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected missing config, err=%v", err)
	}

	if code := Run([]string{"peky", "version"}, "test"); code != 0 {
		t.Fatalf("Run exit code = %d", code)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "Global Configuration") {
		t.Fatalf("config missing default header")
	}
}
